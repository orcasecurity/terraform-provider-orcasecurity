package api_client

import (
	"fmt"
)

const dspmPolicyBasePath = "/api/scan_configuration/dspm_policies"

// DSPMPolicyDocument is the selector document of a DSPM data protection policy.
type DSPMPolicyDocument struct {
	SelectorDetectors  []string `json:"selector_detectors"`
	SelectorCategories []string `json:"selector_categories,omitempty"`
	SelectorRegions    []string `json:"selector_regions,omitempty"`
	SelectorIndustries []string `json:"selector_industries,omitempty"`
	SelectorTags       []string `json:"selector_tags,omitempty"`
	SelectorCountries  []string `json:"selector_countries,omitempty"`
}

// DSPMPolicy is a data protection policy from /api/scan_configuration/dspm_policies.
// Tags and AdvancedSettings have no omitempty on purpose: the server expects
// them present ([] and {} respectively); callers must set them non-nil.
type DSPMPolicy struct {
	ID               string                 `json:"policy_id,omitempty"`
	OrganizationID   string                 `json:"organization,omitempty"`
	Name             string                 `json:"policy_name"`
	Description      string                 `json:"policy_description"`
	Feature          string                 `json:"feature,omitempty"`
	Tags             []string               `json:"tags"`
	Document         DSPMPolicyDocument     `json:"policy_document"`
	AdvancedSettings map[string]interface{} `json:"advanced_settings"`
	IsDefaultPolicy  bool                   `json:"is_default_policy,omitempty"`
}

// GetDSPMPolicy retrieves one policy. Returns (nil, nil) on 404 so the
// resource Read can RemoveResource on remote drift.
func (client *APIClient) GetDSPMPolicy(id string) (*DSPMPolicy, error) {
	resp, err := client.Get(fmt.Sprintf("%s/%s", dspmPolicyBasePath, id))
	if resp != nil && resp.StatusCode() == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	type responseType struct {
		Data DSPMPolicy `json:"data"`
	}
	response := responseType{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) CreateDSPMPolicy(data DSPMPolicy) (*DSPMPolicy, error) {
	resp, err := client.Post(dspmPolicyBasePath, data)
	if err != nil {
		return nil, err
	}

	type responseType struct {
		Data DSPMPolicy `json:"data"`
	}
	response := responseType{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateDSPMPolicy(id string, data DSPMPolicy) (*DSPMPolicy, error) {
	resp, err := client.Put(fmt.Sprintf("%s/%s", dspmPolicyBasePath, id), data)
	if err != nil {
		return nil, err
	}

	type responseType struct {
		Data DSPMPolicy `json:"data"`
	}
	response := responseType{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteDSPMPolicy(id string) error {
	_, err := client.Delete(fmt.Sprintf("%s/%s", dspmPolicyBasePath, id))
	return err
}
