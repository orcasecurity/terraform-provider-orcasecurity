package api_client

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

type WebhookExternalServiceConfig = ConfigEnvelope[WebhookResourceConfig]

func (client *APIClient) CreateWebhookConfig(payload WebhookExternalServiceConfig) (*WebhookExternalServiceConfig, error) {
	return CreateExternalServiceConfig[WebhookResourceConfig](client, WebhookConfigServiceName, payload)
}

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
