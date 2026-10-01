package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSavedSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trip")
	handler := savedSelectionHandler(path)
	var etag string
	request := func(method, body string) *httptest.ResponseRecorder {
		response := requestSavedSelection(handler, method, body, etag)
		if response.Code == http.StatusOK || response.Code == http.StatusNoContent {
			require.NotEmpty(t, response.Header().Get("ETag"))
			etag = response.Header().Get("ETag")
		}
		return response
	}

	response := request(http.MethodGet, "")
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.Equal(t, "text/plain; charset=utf-8", response.Header().Get("Content-Type"))
	require.Empty(t, response.Body.String())
	emptyETag := etag
	text := "# summer\nref: sun\nreq: warm\n  Passport\n"
	require.Equal(t, http.StatusNoContent, request(http.MethodPut, text).Code)
	require.NotEqual(t, emptyETag, etag)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, text, string(data))
	require.Equal(t, text, request(http.MethodGet, "").Body.String())

	for _, tc := range []struct {
		name   string
		body   string
		status int
	}{
		{"invalid syntax", "ref:", http.StatusBadRequest},
		{"invalid encoding", "\xff", http.StatusBadRequest},
		{"too large", strings.Repeat("x", (1<<20)+1), http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.status, request(http.MethodPut, tc.body).Code)
			require.Equal(t, text, request(http.MethodGet, "").Body.String())
		})
	}
	response = request(http.MethodPost, "replacement")
	require.Equal(t, http.StatusMethodNotAllowed, response.Code)
	require.Equal(t, "GET, PUT", response.Header().Get("Allow"))
	require.Equal(t, http.StatusNoContent, request(http.MethodPut, "").Code)
	require.Empty(t, request(http.MethodGet, "").Body.String())
	require.Equal(t, emptyETag, etag, "ETags describe the exact file contents")
}

func TestSavedSelectionDisabledOrUnavailable(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		for _, tc := range []struct {
			path      string
			getStatus int
			putStatus int
		}{
			{"", http.StatusNotFound, http.StatusNotFound},
			{filepath.Join(t.TempDir(), "missing", "trip"), http.StatusInternalServerError, http.StatusPreconditionFailed},
			{t.TempDir(), http.StatusInternalServerError, http.StatusInternalServerError},
		} {
			response := requestSavedSelection(savedSelectionHandler(tc.path), method, "Passport", `"previous"`)
			status := tc.getStatus
			if method == http.MethodPut {
				status = tc.putStatus
			}
			require.Equal(t, status, response.Code)
		}
	}
}

func requestSavedSelection(handler http.Handler, method, body, etag string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/saved-selection/", strings.NewReader(body))
	if method == http.MethodPut && etag != "" {
		request.Header.Set("If-Match", etag)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestSavedSelectionRejectsMissingAndStaleVersions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trip")
	handler := savedSelectionHandler(path)
	initial := requestSavedSelection(handler, http.MethodGet, "", "").Header().Get("ETag")
	response := requestSavedSelection(handler, http.MethodPut, "Passport", initial)
	require.Equal(t, http.StatusNoContent, response.Code)
	latest := response.Header().Get("ETag")

	for _, tc := range []struct {
		etag   string
		status int
	}{
		{"", http.StatusPreconditionRequired},
		{initial, http.StatusPreconditionFailed},
		{"W/" + latest, http.StatusPreconditionFailed},
		{"*", http.StatusPreconditionFailed},
	} {
		response := requestSavedSelection(handler, http.MethodPut, "Towel", tc.etag)
		require.Equal(t, tc.status, response.Code)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "Passport", string(data))
	}

	// The comparison reads the file, so changes made outside the UI are noticed.
	require.NoError(t, os.WriteFile(path, []byte("Passport\n# edited externally"), 0600))
	require.Equal(t, http.StatusPreconditionFailed, requestSavedSelection(handler, http.MethodPut, "Towel", latest).Code)
	response = requestSavedSelection(handler, http.MethodGet, "", "")
	require.Equal(t, "Passport\n# edited externally", response.Body.String())
	latest = response.Header().Get("ETag")
	require.Equal(t, http.StatusNoContent, requestSavedSelection(handler, http.MethodPut, "Towel", latest).Code)

	// Deletion is a conflict even when the previous file was empty.
	response = requestSavedSelection(handler, http.MethodGet, "", "")
	response = requestSavedSelection(handler, http.MethodPut, "", response.Header().Get("ETag"))
	require.Equal(t, http.StatusNoContent, response.Code)
	require.NoError(t, os.Remove(path))
	require.Equal(t, http.StatusPreconditionFailed, requestSavedSelection(handler, http.MethodPut, "Towel", response.Header().Get("ETag")).Code)
	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist, "a rejected save must not recreate the file")
}

func TestSimultaneousSavesCannotBothReplaceTheSameVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trip")
	handler := savedSelectionHandler(path)
	etag := requestSavedSelection(handler, http.MethodGet, "", "").Header().Get("ETag")
	start := make(chan struct{})
	responses := make([]*httptest.ResponseRecorder, 2)
	texts := []string{"Passport", "Towel"}
	var writers sync.WaitGroup
	for i, text := range texts {
		writers.Go(func() {
			<-start
			responses[i] = requestSavedSelection(handler, http.MethodPut, text, etag)
		})
	}
	close(start)
	writers.Wait()
	require.ElementsMatch(t, []int{http.StatusNoContent, http.StatusPreconditionFailed}, []int{responses[0].Code, responses[1].Code})
	loaded := requestSavedSelection(handler, http.MethodGet, "", "")
	for i, response := range responses {
		if response.Code == http.StatusNoContent {
			require.Equal(t, texts[i], loaded.Body.String())
			require.Equal(t, response.Header().Get("ETag"), loaded.Header().Get("ETag"))
		}
	}
}
