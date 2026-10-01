package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"unicode/utf8"

	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/storage/file"
)

func savedSelectionHandler(path string) http.Handler {
	// Serialize reads and compare-and-save operations for this server's file.
	// External editors do not participate in this process-local lock.
	var mu sync.Mutex
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if path == "" {
			http.Error(w, "No selection file configured", http.StatusNotFound)
			return
		}
		selection := file.Selection(path)
		switch r.Method {
		case http.MethodGet:
			mu.Lock()
			defer mu.Unlock()
			data, err := selection.Load()
			if err != nil {
				http.Error(w, fmt.Sprintf("Cannot load selection: %v", err), http.StatusInternalServerError)
				return
			}
			if !utf8.Valid(data) {
				http.Error(w, "Selection file must contain UTF-8 text", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("ETag", selectionETag(data))
			// A disconnected client cannot be sent a second response.
			_, _ = w.Write(data)
		case http.MethodPut:
			match := r.Header.Get("If-Match")
			if match == "" {
				http.Error(w, "Load the selection first and send its ETag in If-Match", http.StatusPreconditionRequired)
				return
			}
			data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
			if err != nil {
				status := http.StatusBadRequest
				var tooLarge *http.MaxBytesError
				if errors.As(err, &tooLarge) {
					status = http.StatusRequestEntityTooLarge
				}
				http.Error(w, fmt.Sprintf("Cannot read selection: %v", err), status)
				return
			}
			if !utf8.Valid(data) {
				http.Error(w, "Selection must contain UTF-8 text", http.StatusBadRequest)
				return
			}
			if _, err := list.ParseContentsDefinition(bytes.NewReader(data)); err != nil {
				http.Error(w, fmt.Sprintf("Cannot parse selection: %v", err), http.StatusBadRequest)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			// PUT must not recreate a file that was deleted since the last load.
			current, err := os.ReadFile(path)
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, "The saved selection was deleted. Load it again before saving.", http.StatusPreconditionFailed)
				return
			}
			if err != nil {
				http.Error(w, fmt.Sprintf("Cannot read current selection: %v", err), http.StatusInternalServerError)
				return
			}
			if match != selectionETag(current) {
				http.Error(w, "The saved selection has changed. Load it again and review your draft before saving.", http.StatusPreconditionFailed)
				return
			}
			if err := selection.Save(data); err != nil {
				http.Error(w, fmt.Sprintf("Cannot save selection: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("ETag", selectionETag(data))
			w.WriteHeader(http.StatusNoContent)
		default:
			w.Header().Set("Allow", "GET, PUT")
			http.Error(w, "Only GET and PUT supported", http.StatusMethodNotAllowed)
		}
	})
}

func selectionETag(data []byte) string {
	return fmt.Sprintf(`"%x"`, sha256.Sum256(data))
}
