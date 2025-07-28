package api_client

import (
	"errors"
	"fmt"
)

type SystemAlert struct {
	RuleType         string   `json:"rule_type"`
	ID               string   `json:"rule_id,omitempty"`
	Title            string   `json:"title"`
	Name             string   `json:"name"`
	Enabled          bool     `json:"enabled"`
	Custom           bool     `json:"custom"`
	Details          string   `json:"details"`
	Recommendation   string   `json:"recommendation"`
	Labels           []string `json:"labels"`
	Category         string   `json:"category"`
	Score            int      `json:"score"`
	FindingAttribute string   `json:"finding_attribute"`
	Rule             string   `json:"rule"`
	Vendors          []string `json:"vendors"`
}

func (client *APIClient) GetSystemSonarAlert(id string) (*SystemAlert, error) {
	type responseType struct {
		Data SystemAlert `json:"data"`
	}

	if id == "" {
		return nil, errors.New("Cannot fetch rule without an id")
	}

	resp, err := client.Get(fmt.Sprintf("/api/sonar/rules/%s", id))
	if resp.StatusCode() == 400 || resp.StatusCode() == 500 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := responseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	alert := response.Data
	return &alert, nil
}

type ChangeSystemSonarAlertStatusResp struct {
	ID      string `json:"rule_id"`
	Enabled bool   `json:"boolean"`
}

func (client *APIClient) ChangeSystemSonarAlertStatus(data SystemAlert) (*ChangeSystemSonarAlertStatusResp, error) {
	type responseType struct {
		Data ChangeSystemSonarAlertStatusResp `json:"data"`
	}

	// Despite the documentation stating otherwise, we must pass the whole alert rule body to
	// the put request to enable/disable an alert.
	// So we fetch the alert rule's body here, change the enabled boolean, and pass everything back to the API
	instance, err := client.GetSystemSonarAlert(data.ID)
	if err != nil {
		return nil, err
	}

	instance.Enabled = data.Enabled

	resp, err := client.Put(fmt.Sprintf("/api/sonar/rules/status/%s", data.ID), instance)
	if err != nil {
		return nil, err
	}

	response := responseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	alert := &response.Data
	return alert, nil
}
