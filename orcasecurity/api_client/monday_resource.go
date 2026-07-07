package api_client

import (
	"fmt"
)

const MondayResourceServiceName = "monday"

// MondayResourceType is the external_service/resources `type` Orca files Monday token
// credentials under (mirrors the API's ExternalServiceResource type for Monday).
const MondayResourceType = "token"

// MondayResourceData is the `data` block of a Monday external_service/resources entry.
// APIToken is write-only — Orca routes it to SSM and strips it from responses — so it is a
// pointer with omitempty (nil is omitted on the wire). AccountSlug is derived server-side
// from the token and returned on read.
type MondayResourceData struct {
	APIToken    *string `json:"api_token,omitempty"`
	AccountSlug string  `json:"account_slug,omitempty"`
}

type MondayResource struct {
	ID          string             `json:"id,omitempty"`
	Name        string             `json:"name"`
	ServiceName string             `json:"service_name,omitempty"`
	Type        string             `json:"type,omitempty"`
	Data        MondayResourceData `json:"data"`
	CreatedAt   string             `json:"created_at,omitempty"`
	UpdatedAt   string             `json:"updated_at,omitempty"`
}

type mondayResourceSingleResponse struct {
	Status string         `json:"status"`
	Data   MondayResource `json:"data"`
}

func (client *APIClient) CreateMondayResource(payload MondayResource) (*MondayResource, error) {
	payload.ServiceName = MondayResourceServiceName
	payload.Type = MondayResourceType

	resp, err := client.Post("/api/external_service/resources", payload)
	if err != nil {
		return nil, err
	}

	response := mondayResourceSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Monday resource create response: %w", err)
	}
	if response.Data.ID == "" {
		// Some endpoints return the resource at the top level (no status/data envelope).
		direct := MondayResource{}
		if err := resp.ReadJSON(&direct); err == nil && direct.ID != "" {
			return &direct, nil
		}
		return nil, fmt.Errorf("monday resource was not returned by the API")
	}
	return &response.Data, nil
}

// ListMondayResources returns every external_service/resources entry filed under
// service_name=monday in the caller's organisation. Orca exposes no name filter on this
// endpoint, so callers match by name client-side.
func (client *APIClient) ListMondayResources() ([]MondayResource, error) {
	resp, err := client.Get(fmt.Sprintf("/api/external_service/resources?service_name=%s", MondayResourceServiceName))
	if err != nil {
		return nil, err
	}
	type listResponse struct {
		Status string           `json:"status"`
		Data   []MondayResource `json:"data"`
	}
	wrapped := listResponse{}
	if err := resp.ReadJSON(&wrapped); err != nil {
		return nil, fmt.Errorf("failed to decode Monday resource list response: %w", err)
	}
	return wrapped.Data, nil
}

func (client *APIClient) GetMondayResourceByName(name string) (*MondayResource, error) {
	all, err := client.ListMondayResources()
	if err != nil {
		return nil, err
	}
	var matches []MondayResource
	for _, item := range all {
		if item.Name == name {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("multiple Monday resources named %q — provide the ID instead", name)
	}
	return &matches[0], nil
}

func (client *APIClient) GetMondayResource(id string) (*MondayResource, error) {
	resp, err := client.Get(fmt.Sprintf("/api/external_service/resources/%s", id))
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}
	response := mondayResourceSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Monday resource get response: %w", err)
	}
	if response.Data.ID == "" {
		direct := MondayResource{}
		if err := resp.ReadJSON(&direct); err == nil && direct.ID != "" {
			return &direct, nil
		}
		return nil, nil
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateMondayResource(id string, payload MondayResource) (*MondayResource, error) {
	payload.ServiceName = MondayResourceServiceName
	payload.Type = MondayResourceType

	resp, err := client.Put(fmt.Sprintf("/api/external_service/resources/%s", id), payload)
	if err != nil {
		return nil, err
	}
	response := mondayResourceSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Monday resource update response: %w", err)
	}
	if response.Data.ID == "" {
		direct := MondayResource{}
		if err := resp.ReadJSON(&direct); err == nil && direct.ID != "" {
			return &direct, nil
		}
		return nil, fmt.Errorf("monday resource was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteMondayResource(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/external_service/resources/%s", id))
	return err
}
