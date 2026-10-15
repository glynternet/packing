package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glynternet/packing/internal/hosted"
	"github.com/stretchr/testify/require"
)

func TestHostedAccessAndRevocation(t *testing.T) {
	dir := t.TempDir()
	store, err := hosted.Open(dir)
	require.NoError(t, err)
	defer store.Close()
	require.NoError(t, store.Seed(map[string]string{"kit": "Tent"}, "ref: kit"))
	secret, err := store.RotateAccess()
	require.NoError(t, err)
	handler := hostedHandler(store, "https://packing.test", false)
	request := func(method, path, body, origin string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Origin", origin)
		r.Header.Set("X-Packing-Request", "1")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	path := "/api/v1/sessions/" + store.Read().ID
	require.Equal(t, 401, request("GET", path, "", "", nil).Code)
	body, _ := json.Marshal(map[string]string{"secret": secret})
	require.Equal(t, 403, request("POST", "/api/v1/exchange", string(body), "https://evil.test", nil).Code)
	require.Equal(t, 401, request("POST", "/api/v1/exchange", `{"secret":"wrong"}`, "https://packing.test", nil).Code)
	exchange := request("POST", "/api/v1/exchange", string(body), "https://packing.test", nil)
	require.Equal(t, 204, exchange.Code)
	cookies := exchange.Result().Cookies()
	require.Len(t, cookies, 1)
	cookie := cookies[0]
	require.True(t, cookie.Secure)
	require.True(t, cookie.HttpOnly)
	require.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
	require.Equal(t, "__Host-packing", cookie.Name)
	require.Equal(t, 200, request("GET", path, "", "", cookie).Code)
	for _, legacy := range []string{"/groups/", "/selection/", "/simplify/", "/saved-selection/"} {
		require.Equal(t, 404, request("GET", legacy, "", "", cookie).Code)
	}
	disk, err := os.ReadFile(filepath.Join(dir, "state.json"))
	require.NoError(t, err)
	require.NotContains(t, string(disk), secret)
	require.NotContains(t, string(disk), cookie.Value)
	// The application shell never includes private trip data or the access secret.
	shell := request("GET", "/sessions/"+store.Read().ID, "", "", nil)
	require.Equal(t, 200, shell.Code)
	require.NotContains(t, shell.Body.String(), "ref: kit")
	require.Equal(t, "no-referrer", shell.Header().Get("Referrer-Policy"))
	_, err = store.RotateAccess()
	require.NoError(t, err)
	require.Equal(t, 401, request("GET", path, "", "", cookie).Code)
	require.Equal(t, 401, request("POST", "/api/v1/exchange", string(body), "https://packing.test", nil).Code)
}
func TestHostedOriginValidation(t *testing.T) {
	for _, origin := range []string{"http://packing.test", "https://packing.test/path", "https://user:secret@packing.test", "https://packing.test?token=x"} {
		_, err := publicOrigin(origin, false)
		require.Error(t, err)
	}
	_, err := publicOrigin("https://packing.test", false)
	require.NoError(t, err)
	_, err = publicOrigin("http://127.0.0.1:8080", true)
	require.NoError(t, err)
	_, err = publicOrigin("http://0.0.0.0:8080", true)
	require.Error(t, err)
}
