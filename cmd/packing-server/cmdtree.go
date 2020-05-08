package main

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/glynternet/packing/internal/load"
	"github.com/glynternet/packing/internal/service"
	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/packing/pkg/cmd"
	"github.com/glynternet/packing/pkg/storage"
	"github.com/glynternet/packing/pkg/storage/file"
	"github.com/glynternet/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func buildCmdTree(logger log.Logger, w io.Writer, rootCmd *cobra.Command) {
	viper.SetEnvPrefix("packing")

	const (
		keyPackingGroups = "groups-dir"
		keyPort          = "port"
	)

	serve := &cobra.Command{
		Use:  "serve",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			groupsDir := strings.TrimSpace(viper.GetString(keyPackingGroups))
			if groupsDir == "" {
				return fmt.Errorf("%s arg cannot be empty", keyPackingGroups)
			}
			if err := logger.Log(log.KV{
				K: "message",
				V: keyPackingGroups + " has been set",
			}, log.KV{
				K: "groupsDir",
				V: groupsDir,
			}); err != nil {
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
			return errors.Wrap(serve(logger, &s, addr), "serving groups service")
		},
	}

	serve.Flags().String(keyPackingGroups, "", "directory containing packing groups")
	serve.Flags().Uint(keyPort, 3865, "port to listen for gRPC")
	cmd.MustBindPFlags(logger, serve)
	rootCmd.AddCommand(serve)
}

func newServer(gss api.GroupsServiceServer) *grpc.Server {
	srv := grpc.NewServer()
	api.RegisterGroupsServiceServer(srv, gss)
	return srv
}

func serve(logger log.Logger, server api.GroupsServiceServer, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrapf(err, "failed to listen at %q", addr)
	}
	if err := logger.Log(
		log.KV{K: "message", V: "Starting server"},
		log.KV{K: "address", V: addr}); err != nil {
		return errors.Wrap(err, "logging")
	}
	sErr := errors.Wrap(newServer(server).Serve(lis), "serving groups service")
	cErr := errors.Wrap(lis.Close(), "closing listener")
	if sErr == nil {
		return cErr
	}
	if cErr != nil {
		_ = logger.Log(
			log.Message("Error closing listener"),
			log.Error(err))
	}
	return sErr
}
