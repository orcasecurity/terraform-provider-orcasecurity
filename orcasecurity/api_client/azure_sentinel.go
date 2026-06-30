package api_client

import (
	"fmt"
	"net/url"
)

const AzureSentinelServiceName = "azure_sentinel"

type AzureSentinelConfig struct {
	LogType     string `json:"log_type,omitempty"`
	PrimaryKey  string `json:"primary_key,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

type AzureSentinelExternalServiceConfig struct {
	ID            string              `json:"id,omitempty"`
	ServiceName   string              `json:"service_name,omitempty"`
	TemplateName  string              `json:"template_name,omitempty"`
	Config        AzureSentinelConfig `json:"config"`
	IsEnabled     bool                `json:"is_enabled"`
	IsDefault     bool                `json:"is_default"`
	BusinessUnits []string            `json:"business_units,omitempty"`
	CreatedAt     string              `json:"created_at,omitempty"`
	UpdatedAt     string              `json:"updated_at,omitempty"`
}

type azureSentinelSingleResponse struct {
	Status string                            `json:"status"`
	Data   AzureSentinelExternalServiceConfig `json:"data"`
}

type azureSentinelListResponse struct {
	Status string                              `json:"status"`
	Data   []AzureSentinelExternalServiceConfig `json:"data"`
}

func (client *APIClient) CreateAzureSentinelConfig(payload AzureSentinelExternalServiceConfig) (*AzureSentinelExternalServiceConfig, error) {
	payload.ServiceName = AzureSentinelServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := azureSentinelSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Azure Sentinel create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Azure Sentinel integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetAzureSentinelConfig(templateName string) (*AzureSentinelExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		AzureSentinelServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := azureSentinelListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Azure Sentinel list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdateAzureSentinelConfig(templateName string, payload AzureSentinelExternalServiceConfig) (*AzureSentinelExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		AzureSentinelServiceName, url.QueryEscape(templateName),
	)

	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
	}
	cfg := map[string]interface{}{}
	if payload.Config.LogType != "" {
		cfg["log_type"] = payload.Config.LogType
	}
	if payload.Config.PrimaryKey != "" {
		cfg["primary_key"] = payload.Config.PrimaryKey
	}
	if payload.Config.WorkspaceID != "" {
		cfg["workspace_id"] = payload.Config.WorkspaceID
	}
	body["config"] = cfg
	if payload.BusinessUnits != nil {
		body["business_units"] = payload.BusinessUnits
	}

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := azureSentinelSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Azure Sentinel update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Azure Sentinel integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteAzureSentinelConfig(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		AzureSentinelServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
