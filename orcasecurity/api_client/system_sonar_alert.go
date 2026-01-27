package api_client

import (
	"fmt"
)

type SystemSonarAlert struct {
	RuleID       string  `json:"rule_id,omitempty"`
	RuleType     string  `json:"rule_type,omitempty"`
	Name         string  `json:"name,omitempty"`
	Title        string  `json:"title,omitempty"`
	Category     string  `json:"category,omitempty"`
	Score        float64 `json:"score,omitempty"`
	Enabled      bool    `json:"enabled"`
	Custom       bool    `json:"custom,omitempty"`
	Details      string  `json:"details,omitempty"`
	Rule         string  `json:"rule,omitempty"`
	Organization string  `json:"organization,omitempty"`
}

type SystemSonarAlertStatusRequest struct {
	RuleID   string `json:"rule_id"`
	RuleType string `json:"rule_type"`
	Enabled  bool   `json:"enabled"`
	Custom   bool   `json:"custom"`
}

type SystemSonarAlertStatusResponse struct {
	Version string `json:"version"`
	RuleID  string `json:"rule_id"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

func (client *APIClient) GetSystemSonarAlert(id string) (*SystemSonarAlert, error) {
	type responseType struct {
		Data SystemSonarAlert `json:"data"`
	}

	resp, err := client.Get(fmt.Sprintf("/api/sonar/rules/%s", id))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == 404 {
		return nil, fmt.Errorf("system alert with ID %s not found", id)
	}

	response := responseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

func (client *APIClient) DoesSystemSonarAlertExist(id string) (bool, error) {
	resp, err := client.Head(fmt.Sprintf("/api/sonar/rules/%s", id))
	// Check for network error (nil response)
	if resp == nil {
		return false, err
	}

	// Check for "not found" or server error status codes
	if resp.StatusCode() == 404 || resp.StatusCode() == 500 {
		return false, nil
	}

	// Return any other errors
	if err != nil {
		return false, err
	}

	return true, nil
}

func (client *APIClient) UpdateSystemSonarAlertStatus(id string, ruleType string, enabled bool) (*SystemSonarAlertStatusResponse, error) {
	request := SystemSonarAlertStatusRequest{
		RuleID:   id,
		RuleType: ruleType,
		Enabled:  enabled,
		Custom:   false,
	}

	resp, err := client.Put(fmt.Sprintf("/api/sonar/rules/status/%s", id), request)
	if err != nil {
		return nil, err
	}

	response := SystemSonarAlertStatusResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	return &response, nil
}
