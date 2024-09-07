package api_client

import (
	"fmt"
)

type Group struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	SSOGroup    bool     `json:"sso_group"`
	Users       []string `json:"users"`
}

type groupAPIResponseType struct {
	Data Group `json:"data"`
}

func (client *APIClient) DoesGroupExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/rbac/group/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) GetGroup(id string) (*Group, error) {
	resp, err := client.Get(fmt.Sprintf("/api/rbac/group/%s", id))
	if resp.StatusCode() == 400 || resp.StatusCode() == 500 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := groupAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	group := response.Data
	group.ID = id
	return &group, nil
}

func (client *APIClient) CreateGroup(data Group) (*Group, error) {
	resp, err := client.Post("/api/rbac/group", data)
	if err != nil {
		return nil, err
	}

	response := groupAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateGroup(data Group) (*Group, error) {
	resp, err := client.Put(fmt.Sprintf("/api/rbac/group/%s", data.ID), data)
	if err != nil {
		return nil, err
	}

	response := groupAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteGroup(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/rbac/group/%s", id))
	return err
}
