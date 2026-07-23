package api_client

import (
	"encoding/json"
	"fmt"
)

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

// API returns projects[], not projects_ids; mapped on read.
type ShiftLeftPolicyProject struct {
	ID string `json:"id"`
}

type ShiftLeftPolicyCatalogControls struct {
	Body json.RawMessage
}

func shiftLeftPolicyBasePath(policyType string) string {
	return fmt.Sprintf("/api/shiftleft/%s/policies/", policyType)
}

func shiftLeftPolicyItemPath(policyType, id string) string {
	return fmt.Sprintf("/api/shiftleft/%s/policies/%s/", policyType, id)
}

// Trailing slash required; slashless path 301-redirects.
func shiftLeftPolicyCatalogPath(policyType string) string {
	return fmt.Sprintf("/api/shiftleft/%s/catalog/controls/", policyType)
}

func shiftLeftPolicyProjectsPath(policyType, id string) string {
	return fmt.Sprintf("/api/shiftleft/%s/policies/%s/projects/", policyType, id)
}

// SetShiftLeftPolicyProjects replaces the policy's project associations via the
// dedicated projects endpoint. Unlike the main policy body (where projects_ids
// is omitempty and an empty slice is dropped, making detach-all impossible), this
// body always sends projects_ids explicitly, so an empty slice detaches all.
// The backend rejects detaching a project that would be left with no active
// policy (400), which surfaces to the user.
func (client *APIClient) SetShiftLeftPolicyProjects(policyType, id string, projectIDs []string) error {
	if projectIDs == nil {
		projectIDs = []string{}
	}
	body := struct {
		ProjectsIds []string `json:"projects_ids"`
	}{ProjectsIds: projectIDs}
	_, err := client.Put(shiftLeftPolicyProjectsPath(policyType, id), body)
	return err
}

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

// 404 => (nil, nil). Use GET not HEAD — scm_posture 5xx on HEAD.
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
