package hosted

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPAuthorizationAndWrites(t *testing.T) {
	s, _ := fixture(t)
	handler := API(s, func(r *http.Request) bool { return r.Header.Get("Authorization") == "test-owner" }, "https://packing.test")
	path := "/api/v1/sessions/" + s.Read().ID
	request := func(method, path, body, owner, origin, etag string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Authorization", owner)
		r.Header.Set("Origin", origin)
		r.Header.Set("If-Match", etag)
		r.Header.Set("X-Packing-Request", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	for _, url := range []string{path, path + "/draft", path + "/preview", "/groups/", "/selection/", "/saved-selection/"} {
		for _, owner := range []string{"", "other-owner"} {
			require.Equal(t, 401, request("GET", url, "", owner, "", "").Code)
		}
	}
	for _, url := range []string{"/groups/", "/selection/", "/saved-selection/", "/api/v1/library", "/api/v1/sessions/other"} {
		require.Equal(t, 404, request("GET", url, "", "test-owner", "", "").Code)
	}
	read := request("GET", path, "", "test-owner", "", "")
	require.Equal(t, 200, read.Code)
	require.Equal(t, "no-store", read.Header().Get("Cache-Control"))
	require.NotEmpty(t, read.Header().Get("ETag"))
	m := mutation(s.Read(), "draft")
	m.Text = "Tent\nWater"
	body, _ := json.Marshal(m)
	require.Equal(t, 403, request("PUT", path+"/draft", string(body), "test-owner", "https://evil.test", m.ETag).Code)
	require.Equal(t, 200, request("PUT", path+"/draft", string(body), "test-owner", "https://packing.test", m.ETag).Code)
	require.Equal(t, 200, request("PUT", path+"/draft", string(body), "test-owner", "https://packing.test", m.ETag).Code, "lost-response retry")
	m.ID = randomID()
	body, _ = json.Marshal(m)
	require.Equal(t, 412, request("PUT", path+"/draft", string(body), "test-owner", "https://packing.test", m.ETag).Code)
	m = mutation(s.Read(), "draft")
	m.ETag = ""
	body, _ = json.Marshal(m)
	require.Equal(t, 428, request("PUT", path+"/draft", string(body), "test-owner", "https://packing.test", "").Code)
	require.Equal(t, 400, request("PUT", path+"/draft", "{} {}", "test-owner", "https://packing.test", "").Code)
	require.Equal(t, 422, request("POST", path+"/preview", `{"text":"ref: ../private"}`, "test-owner", "https://packing.test", "").Code)
	require.Equal(t, 200, request("POST", path+"/preview", `{"text":"ref: basics"}`, "test-owner", "https://packing.test", "").Code)
}
