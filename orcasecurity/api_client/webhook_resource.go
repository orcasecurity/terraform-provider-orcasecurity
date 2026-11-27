package api_client

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const WebhookConfigServiceName = "webhook"

// WebhookSecretSetMarker mirrors orca's SECRET_SET_MARKER.
const WebhookSecretSetMarker = "__orca_secret_set__"

// WebhookCustomHeaderValue mirrors the Orca payload shape for custom headers:
// custom_headers is a map keyed by header name whose value is a list of objects with a
// single "custom" field. The list-of-objects shape lets a single header carry multiple
// values when needed.
type WebhookCustomHeaderValue struct {
	Custom string `json:"custom"`
}

// WebhookCustomHeaderValues is the value of one custom header. It decodes the two shapes the
// API emits: a list of {custom} objects, and a legacy plain string (one {custom: s} entry).
// It always serializes as the list.
//
// An alert-field entry ({"orca": "<field>"}) has no "custom" key and decodes as {custom: ""};
// the provider does not model alert-field header values.
type WebhookCustomHeaderValues []WebhookCustomHeaderValue

func (v *WebhookCustomHeaderValues) UnmarshalJSON(data []byte) error {
	// 1. null: the header has no value.
	if string(bytes.TrimSpace(data)) == "null" {
		*v = WebhookCustomHeaderValues{}
		return nil
	}

	// 2. Legacy shape: the value is the header literal itself.
	var plain string
	if err := json.Unmarshal(data, &plain); err == nil {
		*v = WebhookCustomHeaderValues{{Custom: plain}}
		return nil
	}

	// 3. Current shape: a list of {custom} objects.
	var list []WebhookCustomHeaderValue
	if err := json.Unmarshal(data, &list); err == nil {
		*v = list
		return nil
	}

	return fmt.Errorf("webhook custom header: expected a string or a list of {custom} objects, got: %s", string(data))
}

type WebhookResourceConfig struct {
	WebhookURL    string                               `json:"webhook_url"`
	Type          string                               `json:"type"`
	APIKey        string                               `json:"api_key,omitempty"`
	BodyFields    []string                             `json:"body_fields,omitempty"`
	CustomHeaders map[string]WebhookCustomHeaderValues `json:"custom_headers,omitempty"`
}

type WebhookExternalServiceConfig = ConfigEnvelope[WebhookResourceConfig]

func (client *APIClient) GetWebhookConfigByTemplate(templateName string) (*WebhookExternalServiceConfig, error) {
	return GetExternalServiceConfig[WebhookResourceConfig](client, WebhookConfigServiceName, templateName, nil)
}

func (client *APIClient) UpdateWebhookConfig(templateName string, payload WebhookExternalServiceConfig) (*WebhookExternalServiceConfig, error) {
	// The webhook endpoint accepts the whole config block (the sensitive api_key is the only
	// secret and is always re-submitted from state), so send the struct directly rather than
	// composing a per-field map.
	return UpdateExternalServiceConfig[WebhookResourceConfig](client, WebhookConfigServiceName, templateName, BuildUpdateBody(payload, payload.Config, true))
}

func (client *APIClient) DeleteWebhookConfig(templateName string) error {
	return DeleteExternalServiceConfig(client, WebhookConfigServiceName, templateName)
}
