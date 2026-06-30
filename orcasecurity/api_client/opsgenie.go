package api_client

import (
	"fmt"
	"net/url"
)

const OpsgenieServiceName = "opsgenie"

type OpsgenieConfig struct {
	OpsgenieKey string `json:"opsgenie_key,omitempty"`
}

type OpsgenieExternalServiceConfig struct {
	ID            string         `json:"id,omitempty"`
	ServiceName   string         `json:"service_name,omitempty"`
	TemplateName  string         `json:"template_name,omitempty"`
	Config        OpsgenieConfig `json:"config"`
	IsEnabled     bool           `json:"is_enabled"`
	IsDefault     bool           `json:"is_default"`
	BusinessUnits []string       `json:"business_units,omitempty"`
	CreatedAt     string         `json:"created_at,omitempty"`
	UpdatedAt     string         `json:"updated_at,omitempty"`
}

type opsgenieSingleResponse struct {
	Status string                        `json:"status"`
	Data   OpsgenieExternalServiceConfig `json:"data"`
}

type opsgenieListResponse struct {
	Status string                          `json:"status"`
	Data   []OpsgenieExternalServiceConfig `json:"data"`
}

func (client *APIClient) CreateOpsgenieConfig(payload OpsgenieExternalServiceConfig) (*OpsgenieExternalServiceConfig, error) {
	payload.ServiceName = OpsgenieServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := opsgenieSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Opsgenie create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("opsgenie integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetOpsgenieConfig(templateName string) (*OpsgenieExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		OpsgenieServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := opsgenieListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Opsgenie list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdateOpsgenieConfig(templateName string, payload OpsgenieExternalServiceConfig) (*OpsgenieExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		OpsgenieServiceName, url.QueryEscape(templateName),
	)

	// PUT body is partial. Omit empty opsgenie_key so the API keeps the value already in SSM.
	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
	}
	cfg := map[string]interface{}{}
	if payload.Config.OpsgenieKey != "" {
		cfg["opsgenie_key"] = payload.Config.OpsgenieKey
	}
	body["config"] = cfg
	if payload.BusinessUnits != nil {
		body["business_units"] = payload.BusinessUnits
	}

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := opsgenieSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Opsgenie update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("opsgenie integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteOpsgenieConfig(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		OpsgenieServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
