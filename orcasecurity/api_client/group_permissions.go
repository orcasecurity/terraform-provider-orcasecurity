package api_client

import (
	"fmt"
	"time"
)

type GroupPermission struct {
	ID                string             `json:"id,omitempty"`
	Group             GroupInfo          `json:"group"`
	AllCloudAccounts  bool               `json:"all_cloud_accounts"`
	CloudAccounts     []CloudAccountInfo `json:"cloud_accounts"`
	Role              RoleInfo           `json:"role"`
	UserFilters       []string           `json:"user_filters"`
	ShiftleftProjects interface{}        `json:"shiftleft_projects"`
}

type CloudAccountInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
	var allPermissions []GroupPermission
	limit := 100 // A reasonable page size
	startAtIndex := 0

	for {
		resp, err := client.Get(fmt.Sprintf("/api/rbac/access/group?limit=%d&start_at_index=%d", limit, startAtIndex))
		if err != nil {
			return nil, err
		}

		response := groupPermissionsListAPIResponseType{}
		if err = resp.ReadJSON(&response); err != nil {
			return nil, err
		}

		allPermissions = append(allPermissions, response.Data...)

		if len(response.Data) < limit {
			// Last page reached
			break
		}

		startAtIndex += limit
	}

	return allPermissions, nil
}

// GetGroupPermission retrieves a specific group permission by ID
func (client *APIClient) GetGroupPermission(id string) (*GroupPermission, error) {
	const maxRetries = 15
	const delay = 10 * time.Second

	for i := 0; i < maxRetries; i++ {
		permissions, err := client.GetGroupPermissions()
		if err != nil {
			return nil, err
		}

		for _, permission := range permissions {
			if permission.ID == id {
				return &permission, nil
			}
		}

		if i < maxRetries-1 {
			time.Sleep(delay)
		}
	}

	return nil, fmt.Errorf("group permission with ID %s not found after multiple retries", id)
}

// DoesGroupPermissionExist checks if a group permission exists
func (client *APIClient) DoesGroupPermissionExist(id string) (bool, error) {
	_, err := client.GetGroupPermission(id)
	if err != nil {
		return false, nil
	}
	return true, nil
}

type createGroupPermissionPayload struct {
	GroupID           string   `json:"group_id"`
	RoleID            string   `json:"role_id"`
	AllCloudAccounts  bool     `json:"all_cloud_accounts"`
	CloudAccounts     []string `json:"cloud_accounts"`
	UserFilters       []string `json:"user_filters"`
	ShiftleftProjects []string `json:"shiftleft_projects"`
}

// CreateGroupPermission creates a new group permission
func (client *APIClient) CreateGroupPermission(data GroupPermission) (*GroupPermission, error) {
	// group field in the group is empty or nil, throw error
	if data.Group.ID == "" {
		return nil, fmt.Errorf("group ID must be provided")
	}
	// role field in the role is empty or nil, throw error
	if data.Role.ID == "" {
		return nil, fmt.Errorf("role field in the role must be provided")
	}
	// if userfilters is null, make it empty list
	if data.UserFilters == nil {
		data.UserFilters = []string{}
	}
	if data.ShiftleftProjects == nil {
		data.ShiftleftProjects = []string{}
	}

	var cloudAccountsIDs []string
	if data.AllCloudAccounts {
		cloudAccountsIDs = []string{}
	} else if data.CloudAccounts != nil {
		for _, ca := range data.CloudAccounts {
			cloudAccountsIDs = append(cloudAccountsIDs, ca.ID)
		}
	}

	var shiftleftProjectsIDs []string
	if data.ShiftleftProjects != nil {
		if slp, ok := data.ShiftleftProjects.([]string); ok {
			shiftleftProjectsIDs = slp
		}
	}

	payload := createGroupPermissionPayload{
		GroupID:           data.Group.ID,
		RoleID:            data.Role.ID,
		AllCloudAccounts:  data.AllCloudAccounts,
		CloudAccounts:     cloudAccountsIDs,
		UserFilters:       data.UserFilters,
		ShiftleftProjects: shiftleftProjectsIDs,
	}

	resp, err := client.Post("/api/rbac/access/group", payload)
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
