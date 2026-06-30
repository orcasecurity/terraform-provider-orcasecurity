package api_client

import (
	"fmt"
	"net/url"
)

const PagerDutyServiceName = "pagerduty"

type PagerDutyConfig struct {
	IntegrationKey string `json:"integration_key,omitempty"`
}

type PagerDutyExternalServiceConfig struct {
	ID           string          `json:"id,omitempty"`
	ServiceName  string          `json:"service_name,omitempty"`
	TemplateName string          `json:"template_name,omitempty"`
	Config       PagerDutyConfig `json:"config"`
	IsEnabled    bool            `json:"is_enabled"`
	IsDefault    bool            `json:"is_default"`
	CreatedAt    string          `json:"created_at,omitempty"`
	UpdatedAt    string          `json:"updated_at,omitempty"`
}

type pagerDutySingleResponse struct {
	Status string                         `json:"status"`
	Data   PagerDutyExternalServiceConfig `json:"data"`
}

type pagerDutyListResponse struct {
	Status string                           `json:"status"`
	Data   []PagerDutyExternalServiceConfig `json:"data"`
}

func (client *APIClient) CreatePagerDutyConfig(payload PagerDutyExternalServiceConfig) (*PagerDutyExternalServiceConfig, error) {
	payload.ServiceName = PagerDutyServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := pagerDutySingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode PagerDuty create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("pagerduty integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetPagerDutyConfig(templateName string) (*PagerDutyExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		PagerDutyServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := pagerDutyListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode PagerDuty list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdatePagerDutyConfig(templateName string, payload PagerDutyExternalServiceConfig) (*PagerDutyExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		PagerDutyServiceName, url.QueryEscape(templateName),
	)

	// The PUT endpoint accepts a partial body. Omit empty integration_key so the API keeps the
	// value already in SSM when the user did not change it.
	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
	}
	cfg := map[string]interface{}{}
	if payload.Config.IntegrationKey != "" {
		cfg["integration_key"] = payload.Config.IntegrationKey
	}
	body["config"] = cfg

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := pagerDutySingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode PagerDuty update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("pagerduty integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeletePagerDutyConfig(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		PagerDutyServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
