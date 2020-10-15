package http_test

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	ghttp "github.com/glynternet/pkg/http"
	"github.com/stretchr/testify/assert"
)

type mockRequestAuthoriser struct {
	err     error
	request *http.Request
}

func (ra *mockRequestAuthoriser) Authorise(r *http.Request) error {
	ra.request = r
	return ra.err
}

type mockHandler struct {
	request *http.Request
	writer  http.ResponseWriter
}

func (h *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.writer = w
	h.request = r
}

func TestWithAuthoriser(t *testing.T) {
	t.Run("unauthorised", func(t *testing.T) {
		var buf bytes.Buffer
		logger := log.New(&buf, "", 0)
		var next mockHandler
		ra := mockRequestAuthoriser{err: errors.New("auth error")}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "http://any/", nil)

		hf := ghttp.WithAuthoriser(logger, &ra, &next)
		hf(w, r)
		assert.Equal(t, r, ra.request)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, "Unauthorised request: auth error\n", buf.String())
		assert.Nil(t, next.writer)
		assert.Nil(t, next.request)
	})

	t.Run("authorised", func(t *testing.T) {
		var buf bytes.Buffer
		logger := log.New(&buf, "", 0)
		var ra mockRequestAuthoriser
		next := mockHandler{}
		hf := ghttp.WithAuthoriser(logger, &ra, &next)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "http://any/", nil)
		hf(w, r)
		assert.Equal(t, w, next.writer)
		assert.Equal(t, r, next.request)
		assert.Empty(t, buf)
	})
}
