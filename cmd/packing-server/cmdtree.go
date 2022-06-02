package main

import (
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
	"github.com/glynternet/packing/pkg/cmd"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/glynternet/packing/pkg/storage/file"
	"github.com/glynternet/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	viper.SetEnvPrefix("packing")

	const (
		keyPackingGroups = "groups-dir"
		keyPort          = "port"

		defaultGroupsDir = "."
	)

	serve := &cobra.Command{
		Use:  "serve",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			groupsDir := strings.TrimSpace(viper.GetString(keyPackingGroups))
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

			addr := ":" + strconv.FormatUint(uint64(viper.GetInt64(keyPort)), 10)
			return errors.Wrap(serve(logger, s.GetGroups, addr), "serving groups service")
		},
	}

	serve.Flags().String(keyPackingGroups, "", "directory containing packing groups")
	serve.Flags().Uint(keyPort, 3865, "port to listen on")
	cmd.MustBindPFlags(logger, serve)
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

		apiGroups, err := getGroups(contentsDefinition)
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

func writeStaticContent(logger log.Logger, writer http.ResponseWriter, content []byte) {
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write(content); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_ = log.Error(logger, log.Message("Error writing response"), log.ErrorMessage(err))
	}
}
