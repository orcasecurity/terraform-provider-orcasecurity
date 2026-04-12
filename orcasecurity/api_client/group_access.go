package api_client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strings"
)

const (
	apiRBACGroupAccessPath    = "/api/rbac/access/group"
	apiRBACGroupAccessByIDFmt = "/api/rbac/access/group/%s"
)

// GroupAccess maps to POST/PUT /api/rbac/access/group payloads.
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

// GetGroupAccess fetches a group–role assignment by its Orca assignment id.
func (client *APIClient) GetGroupAccess(id string) (*GroupAccess, error) {
	resp, err := client.Get(fmt.Sprintf(apiRBACGroupAccessByIDFmt, id))
	if err != nil {
		if strings.Contains(err.Error(), "status: 404") {
			return nil, nil
		}
		return nil, err
	}
	body := resp.Body()

	var wrapped groupAccessAPIResponseType
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, err
	}
	out := wrapped.Data
	if out.ID == "" {
		out.ID = id
	}
	return &out, nil
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

// ListGroupAccessForGroup calls GET /api/rbac/access/group?group_id=… and returns rows whose
// nested group id equals groupID (the API may include other groups in the payload).
func (client *APIClient) ListGroupAccessForGroup(groupID string) ([]GroupAccess, error) {
	q := url.Values{}
	q.Set("group_id", groupID)
	path := apiRBACGroupAccessPath + "?" + q.Encode()
	resp, err := client.Get(path)
	if err != nil {
		return nil, err
	}
	body := resp.Body()
	var envelope struct {
		Data []groupAccessListRow `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	out := make([]GroupAccess, 0, len(envelope.Data))
	for _, row := range envelope.Data {
		ga := groupAccessFromListRow(row)
		if ga.GroupID != groupID {
			continue
		}
		out = append(out, ga)
	}
	return out, nil
}

// FindGroupAccess resolves an assignment by id via GET; if that returns 404, it lists
// assignments for want.GroupID and returns the row matching role and scope (same as want).
// want.ID is used only to break ties when multiple API rows match the same scope.
func (client *APIClient) FindGroupAccess(assignmentID string, want GroupAccess) (*GroupAccess, error) {
	byID, err := client.GetGroupAccess(assignmentID)
	if err != nil {
		return nil, err
	}
	if byID != nil {
		return byID, nil
	}
	if want.GroupID == "" {
		return nil, nil
	}
	list, err := client.ListGroupAccessForGroup(want.GroupID)
	if err != nil {
		return nil, err
	}
	want.ID = assignmentID
	return pickMatchingGroupAccess(list, want.GroupID, want), nil
}

// UpdateGroupAccess updates an existing assignment (same payload shape as create).
func (client *APIClient) UpdateGroupAccess(data GroupAccess) (*GroupAccess, error) {
	if data.ID == "" {
		return nil, fmt.Errorf("update group access: id is required")
	}
	resp, err := client.Put(fmt.Sprintf(apiRBACGroupAccessByIDFmt, data.ID), data)
	if err != nil {
		return nil, err
	}
	body := resp.Body()

	var wrapped groupAccessAPIResponseType
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, err
	}
	out := wrapped.Data
	if out.ID == "" {
		out.ID = data.ID
	}
	return &out, nil
}

// DeleteGroupAccess removes a group–role assignment.
func (client *APIClient) DeleteGroupAccess(id string) error {
	_, err := client.Delete(fmt.Sprintf(apiRBACGroupAccessByIDFmt, id))
	if err != nil && strings.Contains(err.Error(), "status: 404") {
		return nil
	}
	return err
}
