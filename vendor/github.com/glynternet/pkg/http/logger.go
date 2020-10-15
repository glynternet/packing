package http

import (
	"log"
	"net/http"
	"time"
)

// LoggingHandler returns a simple Handler that wraps a http.handler and logs any incoming request
func LoggingHandler(logger *log.Logger, inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		logger.Printf(
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}
