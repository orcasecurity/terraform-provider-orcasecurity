package api_client

import (
	"fmt"
)

type WidgetConfig struct {
	ID   string `json:"id"`
	Size string `json:"size"`
}

type CustomDashboardExtraParameters struct {
	Description   string         `json:"description"`
	WidgetsConfig []WidgetConfig `json:"widgets_config"`
}

type CustomDashboard struct {
	ID                string                         `json:"preference_id"`
	Name              string                         `json:"name"`
	FilterData        map[string]interface{}         `json:"filter_data"`
	OrganizationLevel bool                           `json:"organization_level"`
	ViewType          string                         `json:"view_type"`
	ExtraParameters   CustomDashboardExtraParameters `json:"extra_params"`
}

type customDashboardAPIResponseType struct {
	Data CustomDashboard `json:"data"`
}

func (client *APIClient) DoesCustomDashboardExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/user_preferences/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) GetCustomDashboard(id string) (*CustomDashboard, error) {
	resp, err := client.Get(fmt.Sprintf("/api/user_preferences/%s", id))
	if resp.StatusCode() == 400 || resp.StatusCode() == 500 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := customDashboardAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	customDashboard := response.Data
	customDashboard.ID = id
	return &customDashboard, nil
}

func (client *APIClient) CreateCustomDashboard(data CustomDashboard) (*CustomDashboard, error) {
	resp, err := client.Post("/api/user_preferences", data)
	if err != nil {
		return nil, err
	}

	response := customDashboardAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}

	return &response.Data, nil
}

func (client *APIClient) UpdateCustomDashboard(data CustomDashboard) (*CustomDashboard, error) {
	resp, err := client.Put(fmt.Sprintf("/api/user_preferences/%s", data.ID), data)
	if err != nil {
		return nil, err
	}

	response := customDashboardAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteCustomDashboard(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/user_preferences/%s", id))
	return err
}
