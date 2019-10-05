package client

import (
	"context"
	"io"

	"github.com/glynternet/packing/pkg/api/build"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// GetGraph fetches the graph for the given seed
func GetGraph(ctx context.Context, conn *grpc.ClientConn, seed api.ContentsDefinition) ([]api.Group, error) {
	groups, err := api.NewGroupsServiceClient(conn).GetGroups(ctx, &seed)
	if err != nil {
		return nil, errors.Wrap(err, "getting groups")
	}
	var gs []api.Group
	for {
		group, err := groups.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "receiving group")
		}
		gs = append(gs, *group)
	}
	return gs, nil
}
