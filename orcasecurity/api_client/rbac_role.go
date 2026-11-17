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

// builtInShiftleftRoleNames are the roles the UI defaults to the Shift Left global
// scope. Role names are unique per organization, so a custom role cannot shadow one.
var builtInShiftleftRoleNames = map[string]struct{}{
	"Shiftleft Administrator": {},
	"Shiftleft User":          {},
	"Shiftleft Alert Manager": {},
}

// IsBuiltInShiftleftRole reports whether roleID is one of the built-in Shift Left roles.
func (client *APIClient) IsBuiltInShiftleftRole(roleID string) (bool, error) {
	roles, err := client.ListRBACRoles()
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if role.ID == roleID {
			_, ok := builtInShiftleftRoleNames[role.Name]
			return ok, nil
		}
	}
	return false, nil
}
