package api_client

import (
	"encoding/json"
	"fmt"
)

// ShiftLeftProjectSummary is the minimal shape needed to enumerate every
// shift-left project for fleet-wide operations (e.g. bulk policy attach).
// The list endpoint returns a much richer object (policies, exceptions,
// config settings, ...); only id/name/key are surfaced here.
type ShiftLeftProjectSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

// ListShiftLeftProjects returns every shift-left project in the
// organization, paging through /api/shiftleft/projects/.
//
// Unlike the other shift-left SCM list endpoints (github installations,
// gitlab groups, ...), this endpoint ignores the `offset` query param
// (confirmed live: offset=0 and offset=5 return identical windows). It
// instead honors `start_at_index`, the same convention used by
// /api/automations (see ListAutomationsV2), so this uses a dedicated
// paging loop rather than getAllScmPages.
func (client *APIClient) ListShiftLeftProjects() ([]ShiftLeftProjectSummary, error) {
	const pageLimit = 50
	var all []ShiftLeftProjectSummary
	for {
		resp, err := client.Get(fmt.Sprintf("/api/shiftleft/projects/?limit=%d&start_at_index=%d", pageLimit, len(all)))
		if err != nil {
			return nil, err
		}

		var env struct {
			TotalItems int                       `json:"total_items"`
			Data       []ShiftLeftProjectSummary `json:"data"`
		}
		if err := json.Unmarshal(resp.Body(), &env); err != nil {
			return nil, err
		}

		all = append(all, env.Data...)
		if len(env.Data) == 0 || len(all) >= env.TotalItems {
			return all, nil
		}
	}
}
