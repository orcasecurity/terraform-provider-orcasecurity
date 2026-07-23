package api_client

import (
	"encoding/json"
	"fmt"
)

// ShiftLeftPolicy represents an AppSec (Shift Left) policy from the API.
type ShiftLeftPolicy struct {
	ID                       string                   `json:"id,omitempty"`
	Name                     string                   `json:"name"`
	Description              string                   `json:"description"`
	Disabled                 bool                     `json:"disabled"`
	WarnMode                 bool                     `json:"warn_mode"`
	PriorityFailureThreshold string                   `json:"priority_failure_threshold"`
	Type                     string                   `json:"type,omitempty"`
	Builtin                  bool                     `json:"builtin,omitempty"`
	ProjectsIds              []string                 `json:"projects_ids,omitempty"`
	Projects                 []ShiftLeftPolicyProject `json:"projects,omitempty"`
	Controls                 json.RawMessage          `json:"controls,omitempty"`
	PolicyData               json.RawMessage          `json:"policy_data,omitempty"`
	Scope                    json.RawMessage          `json:"scope,omitempty"`
	FeatureScope             []string                 `json:"feature_scope,omitempty"`
	CreatedAt                string                   `json:"created_at,omitempty"`
	CreatedBy                string                   `json:"created_by,omitempty"`
	UpdatedAt                string                   `json:"updated_at,omitempty"`
	UpdatedBy                string                   `json:"updated_by,omitempty"`
}

// ShiftLeftPolicyProject is one entry of the read-only projects list the API
// returns describing the projects this policy is attached to. The API returns
// `projects: [{id,...}]`, not `projects_ids`, so it is mapped on read.
type ShiftLeftPolicyProject struct {
	ID string `json:"id"`
}

// ShiftLeftPolicyCatalogControls holds the raw catalog API response.
type ShiftLeftPolicyCatalogControls struct {
	Body json.RawMessage
}

func shiftLeftPolicyBasePath(policyType string) string {
	return fmt.Sprintf("/api/shiftleft/%s/policies/", policyType)
}

func shiftLeftPolicyItemPath(policyType, id string) string {
	return fmt.Sprintf("/api/shiftleft/%s/policies/%s/", policyType, id)
}

// shiftLeftPolicyCatalogPath keeps the trailing slash: the API 301-redirects
// the slashless form, costing an extra round trip per catalog call.
func shiftLeftPolicyCatalogPath(policyType string) string {
	return fmt.Sprintf("/api/shiftleft/%s/catalog/controls/", policyType)
}

// populateProjectsIds fills ProjectsIds from the read-only `projects` array
// the API returns, unless ProjectsIds was already set explicitly.
func (p *ShiftLeftPolicy) populateProjectsIds() {
	if len(p.ProjectsIds) > 0 || len(p.Projects) == 0 {
		return
	}
	ids := make([]string, 0, len(p.Projects))
	for _, proj := range p.Projects {
		ids = append(ids, proj.ID)
	}
	p.ProjectsIds = ids
}

// GetShiftLeftPolicy reads a policy. Returns (nil, nil) when the policy does
// not exist (404) so callers can treat nil as remote drift; any other failure
// is an error. Reads always use GET: some policy type views (notably
// scm_posture) 5xx on HEAD even when the policy exists.
func (client *APIClient) GetShiftLeftPolicy(policyType, id string) (*ShiftLeftPolicy, error) {
	resp, err := client.Get(shiftLeftPolicyItemPath(policyType, id))
	if resp != nil && resp.StatusCode() == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := ShiftLeftPolicy{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	response.populateProjectsIds()
	return &response, nil
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
