package api_client

import (
	"encoding/json"
	"fmt"
)

// ShiftLeftPolicy represents an AppSec (Shift Left) policy from the API.
type ShiftLeftPolicy struct {
	ID                       string          `json:"id,omitempty"`
	Name                     string          `json:"name"`
	Description              string          `json:"description"`
	Disabled                 bool            `json:"disabled"`
	WarnMode                 bool            `json:"warn_mode"`
	PriorityFailureThreshold string          `json:"priority_failure_threshold"`
	Type                     string          `json:"type,omitempty"`
	Builtin                  bool            `json:"builtin,omitempty"`
	ProjectsIds              []string        `json:"projects_ids,omitempty"`
	Controls                 json.RawMessage `json:"controls,omitempty"`
	PolicyData               json.RawMessage `json:"policy_data,omitempty"`
	Scope                    json.RawMessage `json:"scope,omitempty"`
	FeatureScope             []string        `json:"feature_scope,omitempty"`
	CreatedAt                string          `json:"created_at,omitempty"`
	CreatedBy                string          `json:"created_by,omitempty"`
	UpdatedAt                string          `json:"updated_at,omitempty"`
	UpdatedBy                string          `json:"updated_by,omitempty"`
}

// ShiftLeftPolicyCatalogControls holds the raw catalog API response.
type ShiftLeftPolicyCatalogControls struct {
	Body json.RawMessage
}

// ShiftLeftPolicyTypePath returns the API path segment for a policy type.
func ShiftLeftPolicyTypePath(policyType string) string {
	return policyType
}

func shiftLeftPolicyBasePath(policyType string) string {
	return fmt.Sprintf("/api/shiftleft/%s/policies/", ShiftLeftPolicyTypePath(policyType))
}

func shiftLeftPolicyItemPath(policyType, id string) string {
	return fmt.Sprintf("/api/shiftleft/%s/policies/%s/", ShiftLeftPolicyTypePath(policyType), id)
}

func shiftLeftPolicyCatalogPath(policyType string) string {
	return fmt.Sprintf("/api/shiftleft/%s/catalog/controls", ShiftLeftPolicyTypePath(policyType))
}

func (client *APIClient) GetShiftLeftPolicy(policyType, id string) (*ShiftLeftPolicy, error) {
	resp, err := client.Get(shiftLeftPolicyItemPath(policyType, id))
	if err != nil {
		return nil, err
	}

	if !resp.IsOk() {
		return nil, nil
	}

	response := ShiftLeftPolicy{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *APIClient) DoesShiftLeftPolicyExist(policyType, id string) (bool, error) {
	resp, _ := client.Head(shiftLeftPolicyItemPath(policyType, id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) CreateShiftLeftPolicy(policyType string, policy ShiftLeftPolicy) (*ShiftLeftPolicy, error) {
	resp, err := client.Post(shiftLeftPolicyBasePath(policyType), policy)
	if err != nil {
		return nil, err
	}

	response := ShiftLeftPolicy{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *APIClient) UpdateShiftLeftPolicy(policyType, id string, policy ShiftLeftPolicy) (*ShiftLeftPolicy, error) {
	resp, err := client.Put(shiftLeftPolicyItemPath(policyType, id), policy)
	if err != nil {
		return nil, err
	}

	response := ShiftLeftPolicy{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *APIClient) DeleteShiftLeftPolicy(policyType, id string) error {
	_, err := client.Delete(shiftLeftPolicyItemPath(policyType, id))
	return err
}

func (client *APIClient) GetShiftLeftPolicyCatalogControls(policyType string) (*ShiftLeftPolicyCatalogControls, error) {
	resp, err := client.Get(shiftLeftPolicyCatalogPath(policyType))
	if err != nil {
		return nil, err
	}

	response := ShiftLeftPolicyCatalogControls{
		Body: json.RawMessage(resp.Body()),
	}
	return &response, nil
}
