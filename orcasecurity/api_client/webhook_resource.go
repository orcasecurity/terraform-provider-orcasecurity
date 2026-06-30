package api_client

import (
	"fmt"
	"net/url"
)

const WebhookConfigServiceName = "webhook"

// WebhookCustomHeaderValue mirrors the Orca payload shape for custom headers:
// custom_headers is a map keyed by header name whose value is a list of objects with a
// single "custom" field. The list-of-objects shape lets a single header carry multiple
// values when needed.
type WebhookCustomHeaderValue struct {
	Custom string `json:"custom"`
}

type WebhookResourceConfig struct {
	WebhookURL    string                                `json:"webhook_url"`
	Type          string                                `json:"type"`
	APIKey        string                                `json:"api_key,omitempty"`
	BodyFields    []string                              `json:"body_fields,omitempty"`
	CustomHeaders map[string][]WebhookCustomHeaderValue `json:"custom_headers,omitempty"`
}

type WebhookExternalServiceConfig struct {
	ID            string                `json:"id,omitempty"`
	ServiceName   string                `json:"service_name,omitempty"`
	TemplateName  string                `json:"template_name,omitempty"`
	Config        WebhookResourceConfig `json:"config"`
	IsEnabled     bool                  `json:"is_enabled"`
	IsDefault     bool                  `json:"is_default"`
	BusinessUnits []string              `json:"business_units,omitempty"`
	CreatedAt     string                `json:"created_at,omitempty"`
	UpdatedAt     string                `json:"updated_at,omitempty"`
}

type webhookConfigSingleResponse struct {
	Status string                       `json:"status"`
	Data   WebhookExternalServiceConfig `json:"data"`
}

type webhookConfigListResponse struct {
	Status string                         `json:"status"`
	Data   []WebhookExternalServiceConfig `json:"data"`
}

func (client *APIClient) CreateWebhookConfig(payload WebhookExternalServiceConfig) (*WebhookExternalServiceConfig, error) {
	payload.ServiceName = WebhookConfigServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := webhookConfigSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Webhook create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("webhook integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetWebhookConfigByTemplate(templateName string) (*WebhookExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		WebhookConfigServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := webhookConfigListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Webhook list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdateWebhookConfig(templateName string, payload WebhookExternalServiceConfig) (*WebhookExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		WebhookConfigServiceName, url.QueryEscape(templateName),
	)

	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
		"config":     payload.Config,
	}
	if payload.BusinessUnits != nil {
		body["business_units"] = payload.BusinessUnits
	}

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := webhookConfigSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Webhook update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("webhook integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteWebhookConfig(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		WebhookConfigServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
