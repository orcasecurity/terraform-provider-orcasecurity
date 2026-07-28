package api_client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// No /<id> route: id in body. User list is one-shot (ignores pagination params).
const apiRBACUserAccessPath = "/api/rbac/access/user"

type UserAccess struct {
	ID                string   `json:"id,omitempty"`
	AllCloudAccounts  bool     `json:"all_cloud_accounts"`
	RoleID            string   `json:"role_id"`
	UserID            string   `json:"user_id"`
	CloudAccounts     []string `json:"cloud_accounts"`
	ShiftleftProjects []string `json:"shiftleft_projects"`
	UserFilters       []string `json:"user_filters"`
}

type userAccessListRow struct {
	ID               string          `json:"id"`
	AllCloudAccounts bool            `json:"all_cloud_accounts"`
	User             userAccessRef   `json:"user"`
	Role             userAccessRef   `json:"role"`
	CloudAccounts    []userAccessRef `json:"cloud_accounts"`
	UserFilters      []string        `json:"user_filters"`
	ShiftleftRaw     json.RawMessage `json:"shiftleft_projects"`
}

type userAccessRef struct {
	ID string `json:"id"`
}

func userAccessFromListRow(row userAccessListRow) UserAccess {
	cloudIDs := make([]string, 0, len(row.CloudAccounts))
	for _, ca := range row.CloudAccounts {
		if ca.ID != "" {
			cloudIDs = append(cloudIDs, ca.ID)
		}
	}
	return UserAccess{
		ID:                row.ID,
		UserID:            row.User.ID,
		RoleID:            row.Role.ID,
		AllCloudAccounts:  row.AllCloudAccounts,
		CloudAccounts:     cloudIDs,
		ShiftleftProjects: parseShiftleftProjectIDs(row.ShiftleftRaw),
		UserFilters:       row.UserFilters,
	}
}

// parseUserAccessID extracts the assignment id from a create/update response,
// tolerating both the {"data": {...}} envelope and a bare object.
func parseUserAccessID(body []byte) string {
	var envelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Data.ID != "" {
		return envelope.Data.ID
	}
	var direct struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &direct); err == nil {
		return direct.ID
	}
	return ""
}

// CreateUserAccess assigns a role to a user with optional cloud account, Shift
// Left, or user filter (e.g. business unit) scope.
func (client *APIClient) CreateUserAccess(data UserAccess) (*UserAccess, error) {
	resp, err := client.Post(apiRBACUserAccessPath, data)
	if err != nil {
		return nil, err
	}
	id := parseUserAccessID(resp.Body())
	if id == "" {
		return nil, fmt.Errorf("create user access: could not parse assignment id from response: %s", string(resp.Body()))
	}
	out := data
	out.ID = id
	return &out, nil
}

func (client *APIClient) pageAllUserAccess() ([]UserAccess, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(rbacAccessMaxLimit))
	resp, err := client.Get(apiRBACUserAccessPath + "?" + q.Encode())
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Data []userAccessListRow `json:"data"`
	}
	if err := json.Unmarshal(resp.Body(), &envelope); err != nil {
		return nil, err
	}
	out := make([]UserAccess, 0, len(envelope.Data))
	for _, row := range envelope.Data {
		out = append(out, userAccessFromListRow(row))
	}
	return out, nil
}

// ListUserAccessForUser returns the assignments whose nested user id equals userID.
func (client *APIClient) ListUserAccessForUser(userID string) ([]UserAccess, error) {
	all, err := client.pageAllUserAccess()
	if err != nil {
		return nil, err
	}
	out := make([]UserAccess, 0, len(all))
	for _, ua := range all {
		if ua.UserID == userID {
			out = append(out, ua)
		}
	}
	return out, nil
}

// userAccessScopesMatch compares role and scope fields (not assignment id).
func userAccessScopesMatch(want, got UserAccess) bool {
	return want.RoleID == got.RoleID &&
		want.AllCloudAccounts == got.AllCloudAccounts &&
		stringSliceSetEqual(want.CloudAccounts, got.CloudAccounts) &&
		stringSliceSetEqual(want.ShiftleftProjects, got.ShiftleftProjects) &&
		stringSliceSetEqual(want.UserFilters, got.UserFilters)
}

func pickMatchingUserAccess(list []UserAccess, userID string, want UserAccess) *UserAccess {
	var matches []UserAccess
	for _, item := range list {
		if item.UserID != userID {
			continue
		}
		if !userAccessScopesMatch(want, item) {
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

func (client *APIClient) FindUserAccess(assignmentID string, want UserAccess) (*UserAccess, error) {
	var list []UserAccess
	var err error
	if want.UserID == "" {
		list, err = client.pageAllUserAccess()
	} else {
		list, err = client.ListUserAccessForUser(want.UserID)
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
	return pickMatchingUserAccess(list, want.UserID, want), nil
}

// UpdateUserAccess updates an existing assignment. The id is carried in the body.
func (client *APIClient) UpdateUserAccess(data UserAccess) (*UserAccess, error) {
	if data.ID == "" {
		return nil, fmt.Errorf("update user access: id is required")
	}
	if _, err := client.Put(apiRBACUserAccessPath, data); err != nil {
		return nil, err
	}
	refreshed, err := client.FindUserAccess(data.ID, data)
	if err != nil {
		return nil, err
	}
	if refreshed != nil {
		return refreshed, nil
	}
	out := data
	return &out, nil
}

// DeleteUserAccess removes a user–role assignment (id carried in the body).
func (client *APIClient) DeleteUserAccess(id string) error {
	_, err := client.DeleteWithBody(apiRBACUserAccessPath, map[string]string{"id": id})
	if err != nil && strings.Contains(err.Error(), "status: 404") {
		return nil
	}
	return err
}
