package api_client

import (
	"encoding/json"
	"fmt"
)

// scmEnvelope is the common enveloped list response for shift-left SCM endpoints.
type scmEnvelope[T any] struct {
	TotalItems int `json:"total_items"`
	Data       []T `json:"data"`
}

// getAllScmPages fetches every page of an enveloped {total_items,data} list.
// basePath must already include a leading "/api" and no query string.
//
// Uses limit/start_at_index rather than limit/offset: this API family proved
// to ignore the `offset` query param (confirmed live on the projects
// endpoint, see ListShiftLeftProjects), instead honoring `start_at_index`,
// the same convention used by /api/automations (see ListAutomationsV2).
func getAllScmPages[T any](client *APIClient, basePath string) ([]T, error) {
	const pageLimit = 200
	var all []T
	for {
		resp, err := client.Get(fmt.Sprintf("%s?limit=%d&start_at_index=%d", basePath, pageLimit, len(all)))
		if err != nil {
			return nil, err
		}
		var env scmEnvelope[T]
		if err := json.Unmarshal(resp.Body(), &env); err != nil {
			return nil, err
		}
		all = append(all, env.Data...)
		if len(env.Data) == 0 || len(all) >= env.TotalItems {
			return all, nil
		}
	}
}
