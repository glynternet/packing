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
		err := errors.Wrap(err, "getting AllGroups for seed")
		// TODO(glynternet): is this the best thing to do here or should we send a user facing error back?
		log.Println(err)
		return err
	}
	for _, g := range gs {
		if err := srv.Send(&g); err != nil {
			err := errors.Wrapf(err, "sending group %q", g)
			log.Println(err)
			return err
		}
	}
	return nil
}
