package api_client

import (
	"encoding/json"
	"fmt"
)

type AutomationV2 struct {
	ID             string               `json:"id,omitempty"`
	Name           string               `json:"name"`
	BusinessUnits  []string             `json:"business_units"`
	Description    string               `json:"description"`
	Status         string               `json:"status"`
	Filter         AutomationV2Filter   `json:"filter"`
	Actions        []AutomationV2Action `json:"actions"`
	OrganizationID string               `json:"organization,omitempty"`
	EndTime        string               `json:"end_time,omitempty"`
	CreatedAt      string               `json:"created_at,omitempty"`
	UpdatedAt      string               `json:"updated_at,omitempty"`
}

type AutomationV2Filter struct {
	SonarQuery AutomationV2SonarQuery `json:"sonar_query"`
}

type AutomationV2SonarQuery struct {
	Models []string               `json:"models"`
	Type   string                 `json:"type"`
	With   map[string]interface{} `json:"with,omitempty"` // Complex nested filter structure
}

type AutomationV2Action struct {
	ID             string                 `json:"id,omitempty"`
	Type           int32                  `json:"type"`
	Data           map[string]interface{} `json:"data"`
	ExternalConfig *string                `json:"external_config,omitempty"`
}

func (client *APIClient) GetAutomationV2(automationID string) (*AutomationV2, error) {
	resp, err := client.Get(fmt.Sprintf("/api/automations/%s", automationID))
	if err != nil {
		return nil, err
	}

	if !resp.IsOk() {
		return nil, nil
	}

	// API returns data nested in a "data" field
	var response struct {
		Status string       `json:"status"`
		Data   AutomationV2 `json:"data"`
	}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DoesAutomationV2Exist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/automations/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) CreateAutomationV2(automation AutomationV2) (*AutomationV2, error) {
	resp, err := client.Post("/api/automations", automation)
	if err != nil {
		return nil, err
	}

	// API returns data nested in a "data" field
	var response struct {
		Status string       `json:"status"`
		Data   AutomationV2 `json:"data"`
	}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateAutomationV2(ID string, data AutomationV2) (*AutomationV2, error) {
	resp, err := client.Put(fmt.Sprintf("/api/automations/%s", ID), data)
	if err != nil {
		return nil, err
	}

	// API returns data nested in a "data" field
	var response struct {
		Status string       `json:"status"`
		Data   AutomationV2 `json:"data"`
	}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteAutomationV2(ID string) error {
	_, err := client.Delete(fmt.Sprintf("/api/automations/%s", ID))
	return err
}
