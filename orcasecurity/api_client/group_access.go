package api_client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// The endpoint has no /<id> route (GET/PUT/DELETE on /<id> all 404): the id
// travels in the request body, and reads page the collection and filter
// client-side because the server ignores the group_id/id query params.
const (
	apiRBACGroupAccessPath = "/api/rbac/access/group"
	// rbacAccessPageLimit is the page size for the group list, which paginates
	// (default 10 rows) and honours limit + start_at_index.
	rbacAccessPageLimit = 300
	// rbacAccessMaxLimit is a single-request ceiling for the user list, which
	// ignores start_at_index and returns the whole collection at once.
	rbacAccessMaxLimit = 10000
)

type GroupAccess struct {
	ID                string   `json:"id,omitempty"`
	AllCloudAccounts  bool     `json:"all_cloud_accounts"`
	RoleID            string   `json:"role_id"`
	GroupID           string   `json:"group_id"`
	CloudAccounts     []string `json:"cloud_accounts"`
	ShiftleftProjects []string `json:"shiftleft_projects"`
	UserFilters       []string `json:"user_filters"`
}

type groupAccessAPIResponseType struct {
	Data GroupAccess `json:"data"`
}

// CreateGroupAccess assigns a role to a group with optional cloud account, Shift Left, or user filter (e.g. business unit) scope.
func (client *APIClient) CreateGroupAccess(data GroupAccess) (*GroupAccess, error) {
	resp, err := client.Post(apiRBACGroupAccessPath, data)
	if err != nil {
		return nil, err
	}
	body := resp.Body()

	var wrapped groupAccessAPIResponseType
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Data.ID != "" {
		return &wrapped.Data, nil
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Data) > 0 {
		var withID GroupAccess
		if err := json.Unmarshal(envelope.Data, &withID); err == nil && withID.ID != "" {
			return &withID, nil
		}
	}

	var direct GroupAccess
	if err := json.Unmarshal(body, &direct); err == nil && direct.ID != "" {
		return &direct, nil
	}

	return nil, fmt.Errorf("create group access: could not parse assignment id from response: %s", string(body))
}

type groupAccessListRow struct {
	ID               string          `json:"id"`
	AllCloudAccounts bool            `json:"all_cloud_accounts"`
	Group            groupAccessRef  `json:"group"`
	Role             groupAccessRef  `json:"role"`
	CloudAccounts    []cloudAcctRef  `json:"cloud_accounts"`
	UserFilters      []string        `json:"user_filters"`
	ShiftleftRaw     json.RawMessage `json:"shiftleft_projects"`
}

type groupAccessRef struct {
	ID string `json:"id"`
}

type cloudAcctRef struct {
	ID string `json:"id"`
}

func parseShiftleftProjectIDs(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var strs []string
	if err := json.Unmarshal(raw, &strs); err == nil {
		return strs
	}
	var objs []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &objs); err != nil {
		return nil
	}
	out := make([]string, 0, len(objs))
	for _, o := range objs {
		if o.ID != "" {
			out = append(out, o.ID)
		}
	}
	return out
}

func groupAccessFromListRow(row groupAccessListRow) GroupAccess {
	cloudIDs := make([]string, 0, len(row.CloudAccounts))
	for _, ca := range row.CloudAccounts {
		if ca.ID != "" {
			cloudIDs = append(cloudIDs, ca.ID)
		}
	}
	return GroupAccess{
		ID:                row.ID,
		GroupID:           row.Group.ID,
		RoleID:            row.Role.ID,
		AllCloudAccounts:  row.AllCloudAccounts,
		CloudAccounts:     cloudIDs,
		ShiftleftProjects: parseShiftleftProjectIDs(row.ShiftleftRaw),
		UserFilters:       row.UserFilters,
	}
}

func normalizeStringSliceForCompare(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func stringSliceSetEqual(a, b []string) bool {
	return slices.Equal(normalizeStringSliceForCompare(a), normalizeStringSliceForCompare(b))
}

// groupAccessScopesMatch compares role and scope fields (not assignment id).
func groupAccessScopesMatch(want, got GroupAccess) bool {
	return want.RoleID == got.RoleID &&
		want.AllCloudAccounts == got.AllCloudAccounts &&
		stringSliceSetEqual(want.CloudAccounts, got.CloudAccounts) &&
		stringSliceSetEqual(want.ShiftleftProjects, got.ShiftleftProjects) &&
		stringSliceSetEqual(want.UserFilters, got.UserFilters)
}

func pickMatchingGroupAccess(list []GroupAccess, groupID string, want GroupAccess) *GroupAccess {
	var matches []GroupAccess
	for _, item := range list {
		if item.GroupID != groupID {
			continue
		}
		if !groupAccessScopesMatch(want, item) {
			continue
		}
		matches = append(matches, item)
	}
	if len(matches) == 0 {
		return nil
	}
	if want.ID != "" {
		for _, m := range matches {
			if m.ID == want.ID {
				picked := m
				return &picked
			}
		}
	}
	picked := matches[0]
	return &picked
}

// pageAllGroupAccess pages the whole /api/rbac/access/group collection. The
// server ignores the group_id query param and paginates (default 10 rows), so
// every page must be fetched and filtered client-side.
func (client *APIClient) pageAllGroupAccess() ([]GroupAccess, error) {
	var out []GroupAccess
	fetched := 0
	for {
		// Offset by rows received, not page count, so a short page under-fetches
		// safely instead of skipping rows.
		q := url.Values{}
		q.Set("limit", strconv.Itoa(rbacAccessPageLimit))
		q.Set("start_at_index", strconv.Itoa(fetched))
		resp, err := client.Get(apiRBACGroupAccessPath + "?" + q.Encode())
		if err != nil {
			return nil, err
		}
		var envelope struct {
			TotalItems int                  `json:"total_items"`
			Data       []groupAccessListRow `json:"data"`
		}
		if err := json.Unmarshal(resp.Body(), &envelope); err != nil {
			return nil, err
		}
		for _, row := range envelope.Data {
			out = append(out, groupAccessFromListRow(row))
		}
		fetched += len(envelope.Data)
		if len(envelope.Data) == 0 || fetched >= envelope.TotalItems {
			return out, nil
		}
	}
}

// ListGroupAccessForGroup returns the assignments whose nested group id equals groupID.
func (client *APIClient) ListGroupAccessForGroup(groupID string) ([]GroupAccess, error) {
	all, err := client.pageAllGroupAccess()
	if err != nil {
		return nil, err
	}
	out := make([]GroupAccess, 0, len(all))
	for _, ga := range all {
		if ga.GroupID == groupID {
			out = append(out, ga)
		}
	}
	return out, nil
}

// FindGroupAccess returns the row matching assignmentID, falling back to
// role+scope when the id has changed server-side. When want.GroupID is unset
// (import, where only the id is known) it scans the whole collection.
func (client *APIClient) FindGroupAccess(assignmentID string, want GroupAccess) (*GroupAccess, error) {
	var list []GroupAccess
	var err error
	if want.GroupID == "" {
		list, err = client.pageAllGroupAccess()
	} else {
		list, err = client.ListGroupAccessForGroup(want.GroupID)
	}
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		if item.ID == assignmentID {
			picked := item
			return &picked, nil
		}
	}
	want.ID = assignmentID
	return pickMatchingGroupAccess(list, want.GroupID, want), nil
}

// UpdateGroupAccess updates an existing assignment. The id is carried in the
// body; the PUT response nests group/role, so re-read the canonical row.
func (client *APIClient) UpdateGroupAccess(data GroupAccess) (*GroupAccess, error) {
	if data.ID == "" {
		return nil, fmt.Errorf("update group access: id is required")
	}
	if _, err := client.Put(apiRBACGroupAccessPath, data); err != nil {
		return nil, err
	}
	refreshed, err := client.FindGroupAccess(data.ID, data)
	if err != nil {
		return nil, err
	}
	if refreshed != nil {
		return refreshed, nil
	}
	out := data
	return &out, nil
}

// DeleteGroupAccess removes a group–role assignment (id carried in the body).
func (client *APIClient) DeleteGroupAccess(id string) error {
	_, err := client.DeleteWithBody(apiRBACGroupAccessPath, map[string]string{"id": id})
	if err != nil && strings.Contains(err.Error(), "status: 404") {
		return nil
	}
	return err
}
