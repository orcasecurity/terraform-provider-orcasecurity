package api_client

import (
	"fmt"
)

/*type CreatedBy struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}*/

type CustomRole struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	PermissionGroups []string `json:"permission_groups"`
	//ExpirationDate   string   `json:"expiration_date"`
	Description string `json:"description"`
	//CreatedBy        CreatedBy `json:"created_by"`
}

type customRoleAPIResponseType struct {
	Data CustomRole `json:"data"`
}

func (client *APIClient) DoesCustomRoleExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/rbac/roles/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) GetCustomRole(id string) (*CustomRole, error) {
	resp, err := client.Get(fmt.Sprintf("/api/rbac/roles/%s", id))
	if resp.StatusCode() == 400 || resp.StatusCode() == 500 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := customRoleAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	customRole := response.Data
	customRole.ID = id
	return &customRole, nil
}

func (client *APIClient) createCustomRole(data CustomRole) (*CustomRole, error) {
	resp, err := client.PostCreate("/api/rbac/roles", data)
	if err != nil {
		return nil, err
	}

	response := customRoleAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateCustomRole(data CustomRole) (*CustomRole, error) {
	resp, err := client.Put(fmt.Sprintf("/api/rbac/roles/%s", data.ID), data)
	if err != nil {
		return nil, err
	}

	response := customRoleAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteCustomRole(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/rbac/roles/%s", id))
	return err
}

func (client *APIClient) ListCustomRoleIDsByName(name string) (map[string]bool, error) {
	roles, err := client.ListRBACRoles()
	if err != nil {
		return nil, err
	}
	return idsWhere(roles, func(r *RBACRole) string { return r.ID }, func(r *RBACRole) bool { return sameName(r.Name, name) }), nil
}

func (client *APIClient) CustomRoleCreateOp(data CustomRole) CreateOp[CustomRole] {
	return CreateOp[CustomRole]{
		What:   fmt.Sprintf("custom role named %q", data.Name),
		ID:     func(o *CustomRole) string { return o.ID },
		Create: func() (*CustomRole, error) { return client.createCustomRole(data) },
		IDs:    func() (map[string]bool, error) { return client.ListCustomRoleIDsByName(data.Name) },
		Get:    client.GetCustomRole,
	}
}
