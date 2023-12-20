package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/glynternet/packing/pkg/api"
	"github.com/glynternet/pkg/log"
)

// GetGroups fetches the graph for the given seed
func GetGroups(logger log.Logger, addr string, seed api.Contents) ([]api.Group, error) {
	encoded, err := json.Marshal(seed)
	if err != nil {
		return nil, fmt.Errorf("json encoding seed contents: %w", err)
	}
	// without the trailing slash, this gets a 405 status response for being a GET, I guess from some redirection
	resp, err := http.Post(addr+"/groups/", `application/json`, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("making request to groups endpoint: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http error code: %d", resp.StatusCode)
	}
	var gs []api.Group
	if err := json.NewDecoder(resp.Body).Decode(&gs); err != nil {
		if cErr := resp.Body.Close(); cErr != nil {
			_ = log.Error(logger, log.Message("Error closing response body"), log.ErrorMessage(cErr))
		}
		return nil, fmt.Errorf("json decoding response body: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return nil, fmt.Errorf("closing response body: %w", err)
	}
	return gs, nil
}
