package service

import (
	"github.com/glynternet/packing/internal/load"
	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/pkg/log"
	"github.com/pkg/errors"
)

// GroupsService serves groups
type GroupsService struct {
	load.Loader
	Logger log.Logger
}

func (s *GroupsService) GetGroups(seed api.Contents) ([]api.Group, error) {
	gs, err := s.AllGroups(s.Logger, seed)
	if err != nil {
		err := errors.Wrap(err, "getting AllGroups for seed")
		// TODO(glynternet): is this the best thing to do here or should we send a user facing error back?
		_ = s.Logger.Log(
			log.Message("error getting AllGroups for seed"),
			log.ErrorMessage(err),
		)
		return nil, err
	}
	return gs, nil
}
