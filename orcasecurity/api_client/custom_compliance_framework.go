package api_client

import (
	"encoding/json"
	"fmt"
)

// Request types

type CustomComplianceFrameworkTest struct {
	RuleID            string `json:"rule_id"`
	RuleIDInFramework string `json:"rule_id_in_framework"`
}

type CustomComplianceFrameworkSection struct {
	Name  string                          `json:"name"`
	Tests []CustomComplianceFrameworkTest `json:"tests"`
}

type CustomComplianceFrameworkCreateRequest struct {
	Name        string                             `json:"name"`
	Description string                             `json:"description"`
	Sections    []CustomComplianceFrameworkSection `json:"sections"`
}

type CustomComplianceFrameworkUpdateRequest struct {
	Name        string                             `json:"name"`
	Description string                             `json:"description"`
	Sections    []CustomComplianceFrameworkSection `json:"sections"`
}

// Response types

type CustomComplianceFrameworkWriteResponse struct {
	ID          json.Number `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
}

type customComplianceFrameworkWriteAPIResponse struct {
	Data CustomComplianceFrameworkWriteResponse `json:"data"`
}

type CustomComplianceFrameworkReadResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Custom      bool   `json:"custom"`
	Active      bool   `json:"active"`
	IsReady     bool   `json:"is_ready"`
}

type customComplianceFrameworkReadAPIResponse struct {
	Data CustomComplianceFrameworkReadResponse `json:"data"`
}

func (client *APIClient) GetCustomComplianceFramework(id string) (*CustomComplianceFrameworkReadResponse, error) {
	resp, err := client.Get(fmt.Sprintf("/api/compliance/frameworks/%s", id))
	if resp.StatusCode() == 400 || resp.StatusCode() == 500 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := customComplianceFrameworkReadAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

func (client *APIClient) CreateCustomComplianceFramework(data CustomComplianceFrameworkCreateRequest) (*CustomComplianceFrameworkWriteResponse, error) {
	resp, err := client.Post("/api/compliance/frameworks", data)
	if err != nil {
		return nil, err
	}

	response := customComplianceFrameworkWriteAPIResponse{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateCustomComplianceFramework(id string, data CustomComplianceFrameworkUpdateRequest) (*CustomComplianceFrameworkWriteResponse, error) {
	resp, err := client.Put(fmt.Sprintf("/api/compliance/frameworks/%s", id), data)
	if err != nil {
		return nil, err
	}

	response := customComplianceFrameworkWriteAPIResponse{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteCustomComplianceFramework(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/compliance/frameworks/%s", id))
	return err
}
