package api_client

import (
	"fmt"
)

type GroupPermission struct {
	ID                string    `json:"id,omitempty"`
	Group             GroupInfo `json:"group"`
	AllCloudAccounts  bool      `json:"all_cloud_accounts"`
	CloudAccounts     []string  `json:"cloud_accounts"`
	Role              RoleInfo  `json:"role"`
	UserFilters       []string  `json:"user_filters"`
	ShiftleftProjects []string  `json:"shiftleft_projects"`
}

type GroupInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SSOGroup    bool   `json:"sso_group"`
	Group       string `json:"group"`
}

type RoleInfo struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	ExpirationDate *string `json:"expiration_date"`
}

type groupPermissionAPIResponseType struct {
	Data GroupPermission `json:"data"`
}

type groupPermissionsListAPIResponseType struct {
	TotalItems int               `json:"total_items"`
	Data       []GroupPermission `json:"data"`
	Status     string            `json:"status"`
}

type deleteGroupPermissionRequestBody struct {
	ID string `json:"id"`
}

// GetGroupPermissions retrieves all group permissions
func (client *APIClient) GetGroupPermissions() ([]GroupPermission, error) {
	resp, err := client.Get("/api/rbac/access/group")
	if err != nil {
		return nil, err
	}

	response := groupPermissionsListAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

// GetGroupPermission retrieves a specific group permission by ID
func (client *APIClient) GetGroupPermission(id string) (*GroupPermission, error) {
	permissions, err := client.GetGroupPermissions()
	if err != nil {
		return nil, err
	}

	for _, permission := range permissions {
		if permission.ID == id {
			return &permission, nil
		}
	}

	return nil, fmt.Errorf("group permission with ID %s not found", id)
}

// DoesGroupPermissionExist checks if a group permission exists
func (client *APIClient) DoesGroupPermissionExist(id string) (bool, error) {
	_, err := client.GetGroupPermission(id)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// CreateGroupPermission creates a new group permission
func (client *APIClient) CreateGroupPermission(data GroupPermission) (*GroupPermission, error) {
	resp, err := client.Post("/api/rbac/access/group", data)
	if err != nil {
		return nil, err
	}

	response := groupPermissionAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// UpdateGroupPermission updates an existing group permission
func (client *APIClient) UpdateGroupPermission(data GroupPermission) (*GroupPermission, error) {
	// Note: Based on the API examples, there doesn't seem to be a direct PUT endpoint
	// You might need to delete and recreate, or check if there's an update endpoint
	return nil, fmt.Errorf("update operation not supported by API")
}

// DeleteGroupPermissionWithBody deletes a group permission by ID using DELETE with request body
func (client *APIClient) DeleteGroupPermissionWithBody(id string) error {
	deleteBody := deleteGroupPermissionRequestBody{ID: id}

	resp, err := client.DeleteWithBody("/api/rbac/access/group", deleteBody)
	if err != nil {
		return err
	}

	// Check response status if needed
	if resp.StatusCode() != 200 {
		return fmt.Errorf("failed to delete group permission: status %d", resp.StatusCode())
	}

	return nil
}
