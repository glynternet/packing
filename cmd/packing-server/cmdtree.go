package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/glynternet/packing/internal/load"
	"github.com/glynternet/packing/internal/service"
	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/packing/pkg/list"
	"github.com/glynternet/packing/pkg/simplify"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/glynternet/packing/pkg/storage/file"
	"github.com/glynternet/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const index = `<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1">
    <link rel="manifest" href="/manifest.json">
    <title>Packing</title>
    <script src="elm.js"></script>
</head>
<body>
<div id="app"></div>
<script>
const storedState = localStorage.getItem("appState");
const app = Elm.Main.init({
    flags: {state: storedState},
	node: document.getElementById("app"),
});
app.ports.storeState.subscribe(state => {
    localStorage.setItem("appState", state);
    console.log("Stored state");
});
</script>
</body>
</html>
`

//go:embed elm.js
var elmJS []byte

func buildCmdTree(logger log.Logger, _ io.Writer, rootCmd *cobra.Command) {
	const (
		keyPackingGroups = "groups-dir"
		keyPort          = "port"

		defaultGroupsDir = "."
	)

	var (
		groupsDirFlag string
		port          uint
	)

	serve := &cobra.Command{
		Use:   "serve",
		Args:  cobra.NoArgs,
		Short: "Start the packing HTTP server",
		Long: `Start the packing HTTP server.

The server hosts a directory of reusable group files (--groups-dir), where
each file's name is the key used to reference it (ref:<name>). It exposes:

  POST /groups/     expand a selection (JSON api.Contents) into a full set of groups
  POST /selection/  expand a selection (raw text, as a selection file) into groups
  POST /simplify/   minify a selection (raw text), returning the simplified text
                    and the refs/items removed as JSON
  GET  /            the Elm web UI (also /index.html and /elm.js)

Point packing-cli at this server with --server-host / --server-port.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			groupsDir := strings.TrimSpace(groupsDirFlag)
			if groupsDir == "" {
				groupsDir = defaultGroupsDir
				if err := logger.Log(
					log.Message(keyPackingGroups+" not set, using default"),
					log.KV{K: "default", V: defaultGroupsDir}); err != nil {
					return errors.Wrap(err, "logging")
				}
			}
			if err := logger.Log(
				log.Message("Using groups directory"),
				log.KV{K: "groupsDir", V: groupsDir}); err != nil {
				return errors.Wrap(err, "logging")
			}
			s := service.GroupsService{
				Logger: logger,
				Loader: load.Loader{
					ContentsDefinitionGetter: storage.ContentsDefinitionGetter{
						GetReadCloser: file.ReadCloserGetter(groupsDir),
						Logger:        logger,
					},
				}}

			addr := ":" + strconv.FormatUint(uint64(port), 10)
			return errors.Wrap(serve(logger, s.GetGroups, addr), "serving groups service")
		},
	}

	serve.Flags().StringVar(&groupsDirFlag, keyPackingGroups, "", "directory of group files to serve; each filename is its reference key (defaults to the current directory)")
	serve.Flags().UintVar(&port, keyPort, 3865, "port to listen on")
	rootCmd.AddCommand(serve)
}

func serve(logger log.Logger, getGroups func(api.Contents) ([]api.Group, error), addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrapf(err, "failed to listen at %q", addr)
	}
	if err := logger.Log(
		log.KV{K: "message", V: "Starting server"},
		log.KV{K: "address", V: addr}); err != nil {
		return errors.Wrap(err, "logging")
	}

	var serveMux http.ServeMux
	serveMux.HandleFunc("/groups/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			_ = log.Error(logger, log.Message("Unsupported method"), log.KV{K: "url", V: request.URL}, log.KV{K: "method", V: request.Method})
			writer.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = writer.Write([]byte("Only POST supported"))
			return
		}

		_ = logger.Log(log.Message("Handling groups"), log.KV{K: "path", V: request.URL})

		var contentsDefinition api.Contents
		if err := json.NewDecoder(request.Body).Decode(&contentsDefinition); err != nil {
			_ = log.Error(logger, log.Message("Error decoding request body"), log.ErrorMessage(err))
			err = fmt.Errorf("cannot decode request body: %w", err)
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(err.Error()))
			return
		}

		writeGroups(logger, writer, getGroups, contentsDefinition)
	})
	// /selection/ accepts a raw selection in the same text format as a selection
	// file (ref:/req: tags, plain lines as items, # comments) and expands it into
	// the full set of groups. Parsing is done server-side via
	// list.ParseContentsDefinition so the web UI and the CLI share one definition
	// of the format.
	serveMux.HandleFunc("/selection/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			_ = log.Error(logger, log.Message("Unsupported method"), log.KV{K: "url", V: request.URL}, log.KV{K: "method", V: request.Method})
			writer.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = writer.Write([]byte("Only POST supported"))
			return
		}

		_ = logger.Log(log.Message("Handling selection"), log.KV{K: "path", V: request.URL})

		seed, err := list.ParseContentsDefinition(request.Body)
		if err != nil {
			_ = log.Error(logger, log.Message("Error parsing selection body"), log.ErrorMessage(err))
			err = fmt.Errorf("cannot parse selection: %w", err)
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(err.Error()))
			return
		}

		writeGroups(logger, writer, getGroups, seed)
	})
	// /simplify/ accepts a raw selection (same text format as /selection/) and
	// returns the minified selection text plus the refs/items that were removed.
	// The raw body is kept so redundant lines can be dropped in place, preserving
	// comments, ordering and req: lines (simplify.Filter).
	serveMux.HandleFunc("/simplify/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			_ = log.Error(logger, log.Message("Unsupported method"), log.KV{K: "url", V: request.URL}, log.KV{K: "method", V: request.Method})
			writer.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = writer.Write([]byte("Only POST supported"))
			return
		}

		_ = logger.Log(log.Message("Handling simplify"), log.KV{K: "path", V: request.URL})

		body, err := io.ReadAll(request.Body)
		if err != nil {
			_ = log.Error(logger, log.Message("Error reading request body"), log.ErrorMessage(err))
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte(fmt.Errorf("cannot read request body: %w", err).Error()))
			return
		}

		seed, err := list.ParseContentsDefinition(bytes.NewReader(body))
		if err != nil {
			_ = log.Error(logger, log.Message("Error parsing selection body"), log.ErrorMessage(err))
			err = fmt.Errorf("cannot parse selection: %w", err)
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(err.Error()))
			return
		}

		groups, err := getGroups(seed)
		if err != nil {
			_ = log.Error(logger, log.Message("Error getting groups"), log.ErrorMessage(err))
			err = fmt.Errorf("error getting groups: %w", err)
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte(err.Error()))
			return
		}

		res := simplify.Simplify(seed, groups)
		writeSimplifyResponse(logger, writer, simplifyResponse{
			Selection:    simplify.Filter(string(body), res.RemovedRefs, res.RemovedItems),
			RemovedRefs:  nonNil(res.RemovedRefs),
			RemovedItems: nonNil(res.RemovedItems),
		})
	})
	serveMux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeStaticContent(logger, w, []byte(index))
	})
	serveMux.HandleFunc("/index.html", func(w http.ResponseWriter, _ *http.Request) {
		writeStaticContent(logger, w, []byte(index))
	})
	serveMux.HandleFunc("/elm.js", func(w http.ResponseWriter, _ *http.Request) {
		writeStaticContent(logger, w, elmJS)
	})

	sErr := errors.Wrap((&http.Server{
		Addr:    addr,
		Handler: &serveMux,
	}).Serve(lis), "serving everything")
	cErr := lis.Close()
	if sErr == nil {
		return errors.Wrap(cErr, "closing listener")
	}
	if cErr != nil {
		_ = logger.Log(
			log.Message("Error closing listener"),
			log.ErrorMessage(err))
	}
	return sErr
}

// simplifyResponse is the JSON body returned by /simplify/: the minified
// selection text plus the refs and items that were removed.
type simplifyResponse struct {
	Selection    string   `json:"selection"`
	RemovedRefs  []string `json:"removedRefs"`
	RemovedItems []string `json:"removedItems"`
}

func writeSimplifyResponse(logger log.Logger, writer http.ResponseWriter, resp simplifyResponse) {
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		_ = log.Error(logger, log.Message("Error writing json response"), log.ErrorMessage(err))
	}
}

// nonNil returns a non-nil slice so the JSON response encodes an empty list as
// [] rather than null, keeping the web UI's decoder simple.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// writeGroups expands the given seed into its full set of groups and writes them
// to writer as JSON. It is shared by the /groups/ (JSON body) and /selection/
// (text body) handlers, which differ only in how they obtain the seed.
func writeGroups(logger log.Logger, writer http.ResponseWriter, getGroups func(api.Contents) ([]api.Group, error), seed api.Contents) {
	apiGroups, err := getGroups(seed)
	if err != nil {
		_ = log.Error(logger, log.Message("Error getting groups"), log.ErrorMessage(err))
		err = fmt.Errorf("error getting groups: %w", err)
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(err.Error()))
		return
	}

	groups := []api.Group{}
	for _, apiGroup := range apiGroups {
		var refs []string
		for _, key := range apiGroup.Contents.Refs {
			refs = append(refs, key)
		}
		var items []string
		for _, item := range apiGroup.Contents.Items {
			items = append(items, item)
		}
		groups = append(groups, api.Group{
			Name: apiGroup.Name,
			Contents: api.Contents{
				Refs:  refs,
				Items: items,
			},
		})
	}

	if err := json.NewEncoder(writer).Encode(groups); err != nil {
		_ = log.Error(logger, log.Message("Error writing json response"), log.ErrorMessage(err))
	} else {
		_ = logger.Log(log.Message("Successfully served"), log.KV{K: "groups", V: groups})
	}
}

func writeStaticContent(logger log.Logger, writer http.ResponseWriter, content []byte) {
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write(content); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_ = log.Error(logger, log.Message("Error writing response"), log.ErrorMessage(err))
	}
}
