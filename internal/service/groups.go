package service

import (
	"log"

	"github.com/glynternet/packing/internal/load"
	"github.com/glynternet/packing/pkg/api/build"
	"github.com/pkg/errors"
)

// GroupsService serves groups
type GroupsService struct {
	load.Loader
	Logger *log.Logger
}

// GetGroups sends the groups for the seed to the GetGroupsServer
func (s *GroupsService) GetGroups(seed *api.ContentsDefinition, srv api.GroupsService_GetGroupsServer) error {
	gs, err := s.AllGroups(s.Logger, *seed)
	if err != nil {
		return errors.Wrap(err, "getting AllGroups for seed")
	}
	for _, g := range gs {
		if err := srv.Send(&g); err != nil {
			return errors.Wrapf(err, "sending group %q", g)
		}
	}
	return nil
}
