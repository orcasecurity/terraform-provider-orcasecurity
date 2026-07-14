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
