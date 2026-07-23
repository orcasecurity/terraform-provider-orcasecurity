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

	// Policies is the read-only list the API returns describing the attached
	// policies. The API never echoes the write-only `default_policies` flag, so
	// it is derived from whether any built-in policy is attached (see
	// GetShiftLeftProject). omitempty keeps it out of create/update payloads.
	Policies []ShiftLeftProjectPolicy `json:"policies,omitempty"`
}

// ShiftLeftProjectPolicy is a single entry of the read-only policies list the
// API returns for a project. Only the fields the provider needs are modeled.
type ShiftLeftProjectPolicy struct {
	ID      string `json:"id"`
	Builtin bool   `json:"builtin"`
}

// hasBuiltinPolicy reports whether any attached policy is an Orca built-in,
// which is how the provider reconstructs the write-only default_policies flag.
func hasBuiltinPolicy(policies []ShiftLeftProjectPolicy) bool {
	for _, p := range policies {
		if p.Builtin {
			return true
		}
	}
	return false
}

// GetShiftLeftProject returns the project or an error — never (nil, nil):
// the underlying client errors on any non-OK response, including 404.
func (client *APIClient) GetShiftLeftProject(id string) (*ShiftLeftProject, error) {
	resp, err := client.Get(fmt.Sprintf("/api/shiftleft/projects/%s/", id))
	if err != nil {
		return nil, err
	}

	response := ShiftLeftProject{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	// The API does not return default_policies; reconstruct it from the
	// presence of built-in policies so Read does not drift to false.
	response.DefaultPolicies = hasBuiltinPolicy(response.Policies)
	return &response, nil
}

func (client *APIClient) DoesShiftLeftProjectExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/shiftleft/projects/%s/", id))
	return resp.StatusCode() == 200, nil
}

// Project writes invalidate the SCM list cache: ListShiftLeftProjects pages
// through the cached list helper, so a create/update/delete in the same apply
// must not serve stale project pages to later reads.

func (client *APIClient) CreateShiftLeftProject(shift_left_project ShiftLeftProject) (*ShiftLeftProject, error) {
	resp, err := client.Post("/api/shiftleft/projects/", shift_left_project)
	if err != nil {
		return nil, err
	}
	client.invalidateScmListCache()

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
	client.invalidateScmListCache()

	response := ShiftLeftProject{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *APIClient) DeleteShiftLeftProject(ID string) error {
	_, err := client.Delete(fmt.Sprintf("/api/shiftleft/projects/%s/", ID))
	if err == nil {
		client.invalidateScmListCache()
	}
	return err
}
