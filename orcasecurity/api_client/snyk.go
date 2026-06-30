package api_client

import (
	"fmt"
	"net/url"
)

const SnykServiceName = "snyk"

type SnykConfig struct {
	APIToken string `json:"api_token,omitempty"`
	Region   string `json:"region,omitempty"`
}

type SnykExternalServiceConfig struct {
	ID           string     `json:"id,omitempty"`
	ServiceName  string     `json:"service_name,omitempty"`
	TemplateName string     `json:"template_name,omitempty"`
	Config       SnykConfig `json:"config"`
	IsEnabled    bool       `json:"is_enabled"`
	IsDefault    bool       `json:"is_default"`
	CreatedAt    string     `json:"created_at,omitempty"`
	UpdatedAt    string     `json:"updated_at,omitempty"`
}

type snykSingleResponse struct {
	Status string                    `json:"status"`
	Data   SnykExternalServiceConfig `json:"data"`
}

type snykListResponse struct {
	Status string                      `json:"status"`
	Data   []SnykExternalServiceConfig `json:"data"`
}

func (client *APIClient) CreateSnykConfig(payload SnykExternalServiceConfig) (*SnykExternalServiceConfig, error) {
	payload.ServiceName = SnykServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := snykSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Snyk create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("snyk integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetSnykConfig(templateName string) (*SnykExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		SnykServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := snykListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Snyk list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdateSnykConfig(templateName string, payload SnykExternalServiceConfig) (*SnykExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		SnykServiceName, url.QueryEscape(templateName),
	)

	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
	}
	cfg := map[string]interface{}{}
	if payload.Config.APIToken != "" {
		cfg["api_token"] = payload.Config.APIToken
	}
	if payload.Config.Region != "" {
		cfg["region"] = payload.Config.Region
	}
	body["config"] = cfg

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := snykSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Snyk update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("snyk integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteSnykConfig(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		SnykServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
