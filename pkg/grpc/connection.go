package grpc

import (
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func GetGRPCConnection(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	return conn, errors.Wrap(err, "grpc dialling")
}
