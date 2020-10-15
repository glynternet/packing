package http

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/pkg/errors"
)

// AppJSONHandler handles a request and returns a status code, something that
// should be marshalled into the response body and any errors that occur in the
// process
type AppJSONHandler func(*http.Request) (int, interface{}, error)

// ServeHTTP ensures that an AppJSONHandler satisfies the http.Handler
// interface.
// The ResponseWriter should not have been written to when when the
// AppJSONHandler was called, so we marshal our AppJSONHandler's interface{}
// return value into some bytes as JSON
func (ah AppJSONHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, bod, err := ah(r)

	// handle errors
	if err != nil {
		log.Printf(
			"error serving on AppJSONHandler %v. Error: %v - Status: %d (%s) - Request: %+v",
			ah, err, status, http.StatusText(status), r,
		)
		writeError(w, status, err)
		return
	}

	// here, I don't want to write to the writer immediately using a json
	// encoder, in case there is an error in json encoding
	bs, err := json.Marshal(bod)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, wErr := io.WriteString(w, http.StatusText(http.StatusInternalServerError))
		if wErr != nil {
			log.Print(errors.Wrap(wErr, "writing status text to ResponseWriter"))
		}
		log.Print(errors.Wrap(err, "marshalling json response"))
		return
	}

	w.WriteHeader(status)
	w.Header().Set(`Content-Type`, `application/json; charset=UTF-8`)
	_, wErr := w.Write(bs)
	if wErr != nil {
		log.Print(errors.Wrap(wErr, "writing body to ResponseWriter"))
	}
}

func writeError(w http.ResponseWriter, code int, err error) {
	switch code {
	case http.StatusServiceUnavailable:
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		// We can have cases as granular as we like, if we wanted to
		// return custom errors for specific code codes.
		// TODO: if http.StatusInternalServerError is received, we should return bad request and log the error maybe?
	case http.StatusInternalServerError:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	default:
		// Catch any other errors we haven't explicitly handled
		log.Printf("Unhandled error code: %d, from error: %+v", code, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
