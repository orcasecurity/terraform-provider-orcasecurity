package api_client

import (
	"encoding/json"
	"fmt"
)

type ShiftLeftProject struct {
	ID                               string   `json:"id"`
	Name                             string   `json:"name"`
	Description                      string   `json:"description"`
	Key                              string   `json:"key"`
	DefaultPolicies                  bool     `json:"default_policies"`
	SupportCodeComments              string   `json:"support_code_comments_via_cli,omitempty"`
	SupportCveExceptions             string   `json:"support_cve_exceptions_via_cli,omitempty"`
	SupportSecretDetectionSuppresion string   `json:"support_secret_detection_suppression_via_cli,omitempty"`
	GitDefaultBaselineBranch         string   `json:"git_default_baseline_branch,omitempty"`
	PolicyIds                        []string `json:"policies_ids,omitempty"`
	ExceptionIds                     []string `json:"exceptions_ids,omitempty"`
}

func (client *APIClient) GetShiftLeftProject(id string) (*ShiftLeftProject, error) {
	resp, err := client.Get(fmt.Sprintf("/api/shiftleft/projects/%s/", id))
	if err != nil {
		return nil, err
	}

	if !resp.IsOk() {
		return nil, nil
	}

	response := ShiftLeftProject{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *APIClient) DoesShiftLeftProjectExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/shiftleft/projects/%s/", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) CreateShiftLeftProject(shift_left_project ShiftLeftProject) (*ShiftLeftProject, error) {
	resp, err := client.Post("/api/shiftleft/projects/", shift_left_project)
	if err != nil {
		return nil, err
	}

	response := ShiftLeftProject{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *APIClient) UpdateShiftLeftProject(ID string, data ShiftLeftProject) (*ShiftLeftProject, error) {
	resp, err := client.Put(fmt.Sprintf("/api/shiftleft/projects/%s/", ID), data)
	if err != nil {
		return nil, err
	}

	response := ShiftLeftProject{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *APIClient) DeleteShiftLeftProject(ID string) error {
	_, err := client.Delete(fmt.Sprintf("/api/shiftleft/projects/%s/", ID))
	return err
}
