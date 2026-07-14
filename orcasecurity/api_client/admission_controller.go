package api_client

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// Admission Controller management API (Kubernetes admission controller).
// All endpoints live under /api/admission_controller and wrap responses in
// {"status": "...", "data": ...}. DELETE returns 204 with no body.

type AdmissionControllerClusterScopeKind struct {
	// NOTE: the API uses camelCase keys inside cluster_scope.
	APIGroups []string `json:"apiGroups,omitempty"`
	Kinds     []string `json:"kinds"`
	Versions  []string `json:"versions,omitempty"`
}

type AdmissionControllerClusterScope struct {
	Kinds []AdmissionControllerClusterScopeKind `json:"kinds"`
}

type AdmissionControllerControl struct {
	ID              string                          `json:"id,omitempty"`
	Name            string                          `json:"name"`
	Description     *string                         `json:"description,omitempty"`
	TemplateID      string                          `json:"template_id"`
	TemplateName    string                          `json:"template_name,omitempty"`
	ClusterScope    AdmissionControllerClusterScope `json:"cluster_scope"`
	InputParameters json.RawMessage                 `json:"input_parameters,omitempty"`
}

type admissionControllerControlAPIResponse struct {
	Data AdmissionControllerControl `json:"data"`
}

type admissionControllerControlListAPIResponse struct {
	Data []AdmissionControllerControl `json:"data"`
}

// GetAdmissionControllerControl fetches one control. The API has no
// GET /controls/{id} route (it answers 405), so this filters the list
// endpoint by id. Returns nil (no error) when the control does not exist.
func (client *APIClient) GetAdmissionControllerControl(id string) (*AdmissionControllerControl, error) {
	resp, err := client.Get(fmt.Sprintf(
		"/api/admission_controller/controls?ids=%s", url.QueryEscape(id),
	))
	if err != nil {
		return nil, err
	}

	response := admissionControllerControlListAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	if len(response.Data) > 1 {
		return nil, fmt.Errorf("expected one admission controller control for id %s, got %d", id, len(response.Data))
	}
	return &response.Data[0], nil
}

func (client *APIClient) CreateAdmissionControllerControl(data AdmissionControllerControl) (*AdmissionControllerControl, error) {
	data.ID = ""
	resp, err := client.Post("/api/admission_controller/controls", data)
	if err != nil {
		return nil, err
	}

	response := admissionControllerControlAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateAdmissionControllerControl(data AdmissionControllerControl) (*AdmissionControllerControl, error) {
	resp, err := client.Put(fmt.Sprintf("/api/admission_controller/controls/%s", data.ID), data)
	if err != nil {
		return nil, err
	}

	response := admissionControllerControlAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteAdmissionControllerControl(id string) error {
	resp, err := client.Delete(fmt.Sprintf("/api/admission_controller/controls/%s", id))
	if resp != nil && resp.StatusCode() == 404 {
		return nil
	}
	return err
}

type AdmissionControllerPolicy struct {
	ID          string  `json:"id,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsActive    bool    `json:"is_active"`
	// EnforcementAction is "monitor" or "block".
	EnforcementAction string `json:"enforcement_action"`
	// Controls holds control IDs. The API also accepts a "scopes" key here,
	// but this provider never sends it: the policy_assignment resource owns
	// the policy<->scope link and a full-replace PUT carrying scopes would
	// fight it.
	Controls []string `json:"controls"`
}

type admissionControllerPolicyAPIResponse struct {
	Data AdmissionControllerPolicy `json:"data"`
}

func (client *APIClient) GetAdmissionControllerPolicy(id string) (*AdmissionControllerPolicy, error) {
	resp, err := client.Get(fmt.Sprintf("/api/admission_controller/policies/%s", id))
	if resp != nil && resp.StatusCode() == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := admissionControllerPolicyAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) CreateAdmissionControllerPolicy(data AdmissionControllerPolicy) (*AdmissionControllerPolicy, error) {
	data.ID = ""
	resp, err := client.Post("/api/admission_controller/policies", data)
	if err != nil {
		return nil, err
	}

	response := admissionControllerPolicyAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateAdmissionControllerPolicy(data AdmissionControllerPolicy) (*AdmissionControllerPolicy, error) {
	resp, err := client.Put(fmt.Sprintf("/api/admission_controller/policies/%s", data.ID), data)
	if err != nil {
		return nil, err
	}

	response := admissionControllerPolicyAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteAdmissionControllerPolicy(id string) error {
	resp, err := client.Delete(fmt.Sprintf("/api/admission_controller/policies/%s", id))
	if resp != nil && resp.StatusCode() == 404 {
		return nil
	}
	return err
}

// AdmissionControllerScopePolicy is the embedded read-only policy shape the
// API returns on scope reads. Only the ID is consumed (mapped back to the
// resource's policy_ids); other fields are ignored.
type AdmissionControllerScopePolicy struct {
	ID string `json:"id"`
}

// AdmissionControllerScope is exposed in Terraform as
// orcasecurity_admission_controller_policy_assignment ("scope" is the API's
// name for the entity).
type AdmissionControllerScope struct {
	ID               string   `json:"id,omitempty"`
	Name             string   `json:"name"`
	Description      *string  `json:"description,omitempty"`
	CloudAccounts    []string `json:"cloud_accounts"`
	Clusters         []string `json:"clusters"`
	FullOrganization bool     `json:"full_organization"`
	// PolicyIDs is write-only: reads return the embedded Policies instead.
	PolicyIDs []string                         `json:"policy_ids,omitempty"`
	Policies  []AdmissionControllerScopePolicy `json:"policies,omitempty"`
}

type admissionControllerScopeAPIResponse struct {
	Data AdmissionControllerScope `json:"data"`
}

func (client *APIClient) GetAdmissionControllerScope(id string) (*AdmissionControllerScope, error) {
	resp, err := client.Get(fmt.Sprintf("/api/admission_controller/scopes/%s", id))
	if resp != nil && resp.StatusCode() == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := admissionControllerScopeAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) CreateAdmissionControllerScope(data AdmissionControllerScope) (*AdmissionControllerScope, error) {
	data.ID = ""
	data.Policies = nil
	resp, err := client.Post("/api/admission_controller/scopes", data)
	if err != nil {
		return nil, err
	}

	response := admissionControllerScopeAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateAdmissionControllerScope(data AdmissionControllerScope) (*AdmissionControllerScope, error) {
	data.Policies = nil
	resp, err := client.Put(fmt.Sprintf("/api/admission_controller/scopes/%s", data.ID), data)
	if err != nil {
		return nil, err
	}

	response := admissionControllerScopeAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteAdmissionControllerScope(id string) error {
	resp, err := client.Delete(fmt.Sprintf("/api/admission_controller/scopes/%s", id))
	if resp != nil && resp.StatusCode() == 404 {
		return nil
	}
	return err
}

type AdmissionControllerTemplate struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	DisplayName    string   `json:"display_name"`
	Source         string   `json:"source"`
	ControllerType string   `json:"controller_type"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	SupportedKinds []string `json:"supported_kinds"`
}

type admissionControllerTemplateListAPIResponse struct {
	Data []AdmissionControllerTemplate `json:"data"`
}

func (client *APIClient) GetAdmissionControllerTemplates() ([]AdmissionControllerTemplate, error) {
	resp, err := client.Get("/api/admission_controller/templates")
	if err != nil {
		return nil, err
	}

	response := admissionControllerTemplateListAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return response.Data, nil
}
