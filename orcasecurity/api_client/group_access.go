package api_client

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GroupAccess maps to POST/PUT /api/rbac/access/group payloads.
type GroupAccess struct {
	ID                 string   `json:"id,omitempty"`
	AllCloudAccounts   bool     `json:"all_cloud_accounts"`
	RoleID             string   `json:"role_id"`
	GroupID            string   `json:"group_id"`
	CloudAccounts      []string `json:"cloud_accounts"`
	ShiftleftProjects  []string `json:"shiftleft_projects"`
	UserFilters        []string `json:"user_filters"`
}

type groupAccessAPIResponseType struct {
	Data GroupAccess `json:"data"`
}

// CreateGroupAccess assigns a role to a group with optional cloud account, Shift Left, or user filter (e.g. business unit) scope.
func (client *APIClient) CreateGroupAccess(data GroupAccess) (*GroupAccess, error) {
	resp, err := client.Post("/api/rbac/access/group", data)
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
	resp, err := client.Get(fmt.Sprintf("/api/rbac/access/group/%s", id))
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

// UpdateGroupAccess updates an existing assignment (same payload shape as create).
func (client *APIClient) UpdateGroupAccess(data GroupAccess) (*GroupAccess, error) {
	if data.ID == "" {
		return nil, fmt.Errorf("update group access: id is required")
	}
	resp, err := client.Put(fmt.Sprintf("/api/rbac/access/group/%s", data.ID), data)
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
	_, err := client.Delete(fmt.Sprintf("/api/rbac/access/group/%s", id))
	if err != nil && strings.Contains(err.Error(), "status: 404") {
		return nil
	}
	return err
}
