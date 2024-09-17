package api_client

import (
	"fmt"
)

type DiscoveryQuery struct {
	Data map[string]interface{} `json:"query"`
}

type DiscoveryView struct {
	ID         string         `json:"preference_id"`
	Name       string         `json:"name"`
	FilterData DiscoveryQuery `json:"filter_data"`
	//ResultGrouping    []string       `json:"result_grouping"`
	ExtraParameters   map[string]interface{} `json:"extra_params"`
	OrganizationLevel bool                   `json:"organization_level"`
	ViewType          string                 `json:"view_type"`
}

type discoveryViewAPIResponseType struct {
	Data DiscoveryView `json:"data"`
}

func (client *APIClient) DoesDiscoveryViewExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/user_preferences/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) GetDiscoveryView(id string) (*DiscoveryView, error) {
	resp, err := client.Get(fmt.Sprintf("/api/user_preferences/%s", id))
	if resp.StatusCode() == 400 || resp.StatusCode() == 500 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := discoveryViewAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	discoveryView := response.Data
	discoveryView.ID = id
	return &discoveryView, nil
}

func (client *APIClient) CreateDiscoveryView(data DiscoveryView) (*DiscoveryView, error) {
	resp, err := client.Post("/api/user_preferences", data)
	if err != nil {
		return nil, err
	}

	response := discoveryViewAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateDiscoveryView(data DiscoveryView) (*DiscoveryView, error) {
	resp, err := client.Put(fmt.Sprintf("/api/user_preferences/%s", data.ID), data)
	if err != nil {
		return nil, err
	}

	response := discoveryViewAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteDiscoveryView(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/user_preferences/%s", id))
	return err
}
