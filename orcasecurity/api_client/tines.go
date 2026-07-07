package api_client

// Tines is exposed by Orca as a webhook variant: the API stores it under
// service_name="webhook" with config.type="tines". The wrappers below pin the variant so
// callers can treat Tines as its own integration without leaking the webhook plumbing.

const TinesWebhookType = "tines"

func (client *APIClient) CreateTinesConfig(payload WebhookExternalServiceConfig) (*WebhookExternalServiceConfig, error) {
	payload.Config.Type = TinesWebhookType
	return client.CreateWebhookConfig(payload)
}

func (client *APIClient) GetTinesConfig(templateName string) (*WebhookExternalServiceConfig, error) {
	return GetExternalServiceConfig[WebhookResourceConfig](client, WebhookConfigServiceName, templateName, func(e *WebhookExternalServiceConfig) bool {
		return e.Config.Type == TinesWebhookType
	})
}

func (client *APIClient) UpdateTinesConfig(templateName string, payload WebhookExternalServiceConfig) (*WebhookExternalServiceConfig, error) {
	payload.Config.Type = TinesWebhookType
	return client.UpdateWebhookConfig(templateName, payload)
}

func (client *APIClient) DeleteTinesConfig(templateName string) error {
	return client.DeleteWebhookConfig(templateName)
}
