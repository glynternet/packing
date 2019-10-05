package grpc

import (
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// GetGRPCConnection creates a GRPC Connection at the given address
func GetGRPCConnection(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	return conn, errors.Wrap(err, "grpc dialling")
}
