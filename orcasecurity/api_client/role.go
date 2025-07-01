package api_client

import (
	"fmt"
	"net/url"
)

type Role struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	PermissionGroups []string `json:"permission_groups"`
	ExpirationDate   *string  `json:"expiration_date"`
	Description      string   `json:"description"`
	IsCustom         bool     `json:"is_custom"`
	CreatedBy        *user    `json:"created_by"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

func (client *APIClient) GetRoleByName(name string) (*Role, error) {
	resp, err := client.Get(
		fmt.Sprintf("/api/rbac/roles?search=%s&limit=10",
			url.QueryEscape(name),
		),
	)
	if err != nil {
		return nil, err
	}

	type respsoneType struct {
		Data []Role `json:"data"`
	}
	response := respsoneType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("role with name '%s' does not exists", name)
	}
	// There may be partial matches and muliple results, find exact match and filter down to it
	for _, role := range response.Data {
		if role.Name == name {
			return &role, nil
		}
	}
	return nil, fmt.Errorf("role with name '%s' does not exists", name)
}
