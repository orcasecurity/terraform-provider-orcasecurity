package api_client

import (
	"fmt"
)

type customRoleAPIResponseType struct {
	Data Role `json:"data"`
}

func (client *APIClient) DoesCustomRoleExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/rbac/roles/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) GetCustomRole(id string) (*Role, error) {
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

func (client *APIClient) CreateCustomRole(data Role) (*Role, error) {
	resp, err := client.Post("/api/rbac/roles", data)
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

func (client *APIClient) UpdateCustomRole(data Role) (*Role, error) {
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
