package api_client

import (
	"encoding/json"
	"fmt"
)

// RBACRole is one role from GET /api/rbac/role (built-in and org roles).
type RBACRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type rbacRoleListAPIResponse struct {
	Status string     `json:"status"`
	Data   []RBACRole `json:"data"`
}

// ListRBACRoles returns all assignable roles (GET /api/rbac/role).
func (client *APIClient) ListRBACRoles() ([]RBACRole, error) {
	resp, err := client.Get("/api/rbac/role")
	if err != nil {
		return nil, err
	}

	var parsed rbacRoleListAPIResponse
	if err := json.Unmarshal(resp.Body(), &parsed); err != nil {
		return nil, fmt.Errorf("parse rbac role list: %w", err)
	}
	if parsed.Status != "" && parsed.Status != "success" {
		return nil, fmt.Errorf("unexpected rbac role list status: %q", parsed.Status)
	}
	return parsed.Data, nil
}
