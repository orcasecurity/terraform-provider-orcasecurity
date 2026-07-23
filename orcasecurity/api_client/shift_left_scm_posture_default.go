package api_client

import "encoding/json"

// Singleton at /scm_posture/policy/; only disabled and policy_data are writable.
type ScmPostureDefaultPolicy struct {
	ID          string          `json:"id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Disabled    bool            `json:"disabled"`
	PolicyData  json.RawMessage `json:"policy_data,omitempty"`
}

type ScmPostureDefaultPolicyWrite struct {
	Disabled   bool                        `json:"disabled"`
	PolicyData ScmPostureDefaultPolicyData `json:"policy_data"`
}

type ScmPostureDefaultPolicyData struct {
	Controls []ScmPostureControlOverride `json:"controls"`
}

type ScmPostureControlOverride struct {
	ID       string `json:"id"`
	Disabled *bool  `json:"disabled,omitempty"`
	Priority string `json:"priority,omitempty"`
}

const scmPostureDefaultPolicyPath = "/api/shiftleft/scm_posture/policy/"

func (client *APIClient) GetScmPostureDefaultPolicy() (*ScmPostureDefaultPolicy, error) {
	resp, err := client.Get(scmPostureDefaultPolicyPath)
	if err != nil {
		return nil, err
	}
	policy := ScmPostureDefaultPolicy{}
	if err := resp.ReadJSON(&policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

func (client *APIClient) UpdateScmPostureDefaultPolicy(body ScmPostureDefaultPolicyWrite) (*ScmPostureDefaultPolicy, error) {
	resp, err := client.Put(scmPostureDefaultPolicyPath, body)
	if err != nil {
		return nil, err
	}
	policy := ScmPostureDefaultPolicy{}
	if err := resp.ReadJSON(&policy); err != nil {
		return nil, err
	}
	return &policy, nil
}
