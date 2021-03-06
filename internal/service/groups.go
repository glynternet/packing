package service

import (
	"github.com/glynternet/packing/internal/load"
	api "github.com/glynternet/packing/pkg/api/build"
	"github.com/glynternet/pkg/log"
	"github.com/pkg/errors"
)

// GroupsService serves groups
type GroupsService struct {
	load.Loader
	Logger log.Logger
	api.UnimplementedGroupsServiceServer
}

// GetGroups sends the groups for the seed to the GetGroupsServer
func (s *GroupsService) GetGroups(seed *api.ContentsDefinition, srv api.GroupsService_GetGroupsServer) error {
	if seed == nil {
		return errors.New("no seed provided")
	}
	gs, err := s.AllGroups(s.Logger, seed)
	if err != nil {
		err := errors.Wrap(err, "getting AllGroups for seed")
		// TODO(glynternet): is this the best thing to do here or should we send a user facing error back?
		_ = s.Logger.Log(
			log.Message("error getting AllGroups for seed"),
			log.ErrorMessage(err),
		)
		return err
	}
	for _, g := range gs {
		if err := srv.Send(g); err != nil {
			err := errors.Wrapf(err, "sending group %v", g)
			_ = s.Logger.Log(
				log.Message("error getting AllGroups for seed"),
				log.ErrorMessage(err))
			return err
		}
	}
	return nil
}
