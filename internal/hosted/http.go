package hosted

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/simplify"
)

type Authorize func(*http.Request) bool

// API serves only the single selected trip. No legacy filesystem endpoints or
// library/session management APIs are registered in hosted mode.
func API(store *Store, authorize Authorize, origin string) http.Handler {
	slots := make(chan struct{}, 8)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if authorize == nil || !authorize(r) {
			writeFault(w, fault(401, "access_required"))
			return
		}
		if r.Method != "GET" && (r.Header.Get("Origin") != origin || r.Header.Get("X-Packing-Request") != "1") {
			writeFault(w, fault(403, "origin_required"))
			return
		}
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			w.Header().Set("Retry-After", "1")
			writeFault(w, fault(429, "busy"))
			return
		}
		if !store.Ready() {
			writeFault(w, fault(503, "storage_unavailable"))
			return
		}
		trip := store.Read()
		prefix := "/api/v1/sessions/" + trip.ID
		if r.URL.Path != prefix && !strings.HasPrefix(r.URL.Path, prefix+"/") {
			writeFault(w, fault(404, "not_found"))
			return
		}
		action := strings.TrimPrefix(r.URL.Path, prefix)
		if action == "" && r.Method == "GET" {
			w.Header().Set("ETag", ETag(trip, "trip", trip.Revision))
			writeJSON(w, trip)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxText*6+8192)
		if (action == "/preview" || action == "/simplify") && r.Method == "POST" {
			var request struct {
				Text string `json:"text"`
			}
			if err := decode(r, &request); err != nil {
				writeFault(w, err)
				return
			}
			groups, err := store.Preview(request.Text)
			if err != nil {
				writeFault(w, err)
				return
			}
			if action == "/preview" {
				writeJSON(w, groups)
				return
			}
			seed, err := list.ParseContentsDefinition(strings.NewReader(request.Text))
			if err != nil {
				writeFault(w, fault(422, "cannot_resolve_selection"))
				return
			}
			result := simplify.Simplify(seed, groups)
			refs, items := result.RemovedRefs, result.RemovedItems
			if refs == nil {
				refs = []string{}
			}
			if items == nil {
				items = []string{}
			}
			writeJSON(w, map[string]any{"selection": simplify.Filter(request.Text, refs, items), "removedRefs": refs, "removedItems": items})
			return
		}
		kind := ""
		item := ""
		switch {
		case action == "/draft" && r.Method == "PUT":
			kind = "draft"
		case action == "/apply" && r.Method == "POST":
			kind = "apply"
		case strings.HasPrefix(action, "/items/") && r.Method == "PUT":
			kind = "item"
			item = strings.TrimPrefix(action, "/items/")
		default:
			writeFault(w, fault(404, "not_found"))
			return
		}
		var mutation Mutation
		if err := decode(r, &mutation); err != nil {
			writeFault(w, err)
			return
		}
		// Route and conditional header are authoritative. Reject conflicting copies
		// to keep a retry's request digest unambiguous.
		if mutation.Kind != kind || mutation.Item != item || mutation.ETag != r.Header.Get("If-Match") {
			writeFault(w, fault(400, "inconsistent_request"))
			return
		}
		outcome, err := store.Mutate(mutation)
		if err != nil {
			writeFault(w, err)
			return
		}
		// This may be newer than the receipt (a retry or another writer). Clients
		// acknowledge the original outcome separately and accept only newer reads.
		current := store.Read()
		writeJSON(w, struct {
			Outcome Outcome `json:"outcome"`
			Trip    Trip    `json:"trip"`
		}{outcome, current})
	})
}
func decode(r *http.Request, value any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	err := d.Decode(value)
	if err == nil {
		var extra any
		if d.Decode(&extra) != io.EOF {
			err = errors.New("extra JSON")
		}
	}
	if err != nil {
		var max *http.MaxBytesError
		if errors.As(err, &max) {
			return fault(413, "request_too_large")
		}
		return fault(400, "invalid_json")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, value any) { _ = json.NewEncoder(w).Encode(value) }
func writeFault(w http.ResponseWriter, err error) {
	var f *Fault
	if !errors.As(err, &f) {
		f = &Fault{503, "unavailable"}
	}
	w.WriteHeader(f.Status)
	writeJSON(w, map[string]string{"code": f.Code})
}
