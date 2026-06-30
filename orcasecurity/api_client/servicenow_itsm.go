package api_client

import (
	"fmt"
)

const ServiceNowITSMServiceName = "sn_incidents"
const ServiceNowITSMResourceType = "basic"

type ServiceNowITSMData struct {
	Username string  `json:"username"`
	Password *string `json:"password,omitempty"`
}

type ServiceNowITSMResource struct {
	ID          string             `json:"id,omitempty"`
	Name        string             `json:"name"`
	ServiceName string             `json:"service_name"`
	Type        string             `json:"type"`
	HostURL     string             `json:"host_url"`
	Data        ServiceNowITSMData `json:"data"`
	CreatedAt   string             `json:"created_at,omitempty"`
	UpdatedAt   string             `json:"updated_at,omitempty"`
}

type serviceNowITSMSingleResponse struct {
	Status string                 `json:"status"`
	Data   ServiceNowITSMResource `json:"data"`
}

func (client *APIClient) CreateServiceNowITSMResource(payload ServiceNowITSMResource) (*ServiceNowITSMResource, error) {
	payload.ServiceName = ServiceNowITSMServiceName
	payload.Type = ServiceNowITSMResourceType

	resp, err := client.Post("/api/external_service/resources", payload)
	if err != nil {
		return nil, err
	}

	response := serviceNowITSMSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode ServiceNow ITSM create response: %w", err)
	}

	if response.Data.ID == "" {
		// Some endpoints in this API surface return the resource at the top level (no
		// status/data envelope). Try to decode in that shape before giving up.
		direct := ServiceNowITSMResource{}
		if err := resp.ReadJSON(&direct); err == nil && direct.ID != "" {
			return &direct, nil
		}
		return nil, fmt.Errorf("servicenow itsm resource was not returned by the API")
	}

	return &response.Data, nil
}

// ListServiceNowITSMResources returns every external_service/resources entry filed under
// service_name=sn_incidents in the caller's organisation. Use it when looking up an existing
// resource by its human-friendly “name“ (Orca does not expose a name filter on this
// endpoint, so the provider does the match client-side).
func (client *APIClient) ListServiceNowITSMResources() ([]ServiceNowITSMResource, error) {
	resp, err := client.Get(fmt.Sprintf("/api/external_service/resources?service_name=%s", ServiceNowITSMServiceName))
	if err != nil {
		return nil, err
	}

	type listResponse struct {
		Status string                   `json:"status"`
		Data   []ServiceNowITSMResource `json:"data"`
	}
	wrapped := listResponse{}
	if err := resp.ReadJSON(&wrapped); err != nil {
		return nil, fmt.Errorf("failed to decode ServiceNow ITSM list response: %w", err)
	}
	return wrapped.Data, nil
}

func (client *APIClient) GetServiceNowITSMResourceByName(name string) (*ServiceNowITSMResource, error) {
	all, err := client.ListServiceNowITSMResources()
	if err != nil {
		return nil, err
	}
	var matches []ServiceNowITSMResource
	for _, item := range all {
		if item.Name == name {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("multiple ServiceNow ITSM resources named %q — provide the ID instead", name)
	}
	return &matches[0], nil
}

func (client *APIClient) GetServiceNowITSMResource(id string) (*ServiceNowITSMResource, error) {
	resp, err := client.Get(fmt.Sprintf("/api/external_service/resources/%s", id))
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := serviceNowITSMSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode ServiceNow ITSM read response: %w", err)
	}

	if response.Data.ID == "" {
		direct := ServiceNowITSMResource{}
		if err := resp.ReadJSON(&direct); err == nil && direct.ID != "" {
			return &direct, nil
		}
		return nil, nil
	}

	return &response.Data, nil
}

func (client *APIClient) UpdateServiceNowITSMResource(id string, payload ServiceNowITSMResource) (*ServiceNowITSMResource, error) {
	payload.ServiceName = ServiceNowITSMServiceName
	payload.Type = ServiceNowITSMResourceType

	resp, err := client.Put(fmt.Sprintf("/api/external_service/resources/%s", id), payload)
	if err != nil {
		return nil, err
	}

	response := serviceNowITSMSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode ServiceNow ITSM update response: %w", err)
	}

	if response.Data.ID == "" {
		direct := ServiceNowITSMResource{}
		if err := resp.ReadJSON(&direct); err == nil && direct.ID != "" {
			return &direct, nil
		}
		return nil, fmt.Errorf("servicenow itsm resource was not returned by the API")
	}

	return &response.Data, nil
}

func (client *APIClient) DeleteServiceNowITSMResource(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/external_service/resources/%s", id))
	return err
}
