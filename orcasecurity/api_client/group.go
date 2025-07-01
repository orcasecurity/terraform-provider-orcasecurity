package api_client

import (
	"fmt"
	"net/url"
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

type groupListResponse struct {
	Status string `json:"status"`
	Data   struct {
		Groups []Group `json:"groups"`
	} `json:"data"`
}

func (client *APIClient) GetGroupByName(name string) (*Group, error) {
	resp, err := client.Get(
		fmt.Sprintf("/api/rbac/group?search=%s&limit=50",
			url.QueryEscape(name),
		),
	)
	if err != nil {
		return nil, err
	}

	response := groupListResponse{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}

	if len(response.Data.Groups) == 0 {
		return nil, fmt.Errorf("group with name '%s' does not exist", name)
	}

	// Find exact match - there may be partial matches, filter to exact match
	for _, group := range response.Data.Groups {
		if group.Name == name {
			return &group, nil
		}
	}

	return nil, fmt.Errorf("group with name '%s' does not exist", name)
}
