package api_client

// Torq is exposed by Orca as a webhook variant: the API stores it under
// service_name="webhook" with config.type="torq". The wrappers below pin the variant.

const TorqWebhookType = "torq"

func (client *APIClient) CreateTorqConfig(payload WebhookExternalServiceConfig) (*WebhookExternalServiceConfig, error) {
	payload.Config.Type = TorqWebhookType
	return client.CreateWebhookConfig(payload)
}

func (client *APIClient) GetTorqConfig(templateName string) (*WebhookExternalServiceConfig, error) {
	return GetExternalServiceConfig[WebhookResourceConfig](client, WebhookConfigServiceName, templateName, func(e *WebhookExternalServiceConfig) bool {
		return e.Config.Type == TorqWebhookType
	})
}

func (client *APIClient) UpdateTorqConfig(templateName string, payload WebhookExternalServiceConfig) (*WebhookExternalServiceConfig, error) {
	payload.Config.Type = TorqWebhookType
	return client.UpdateWebhookConfig(templateName, payload)
}

func (client *APIClient) DeleteTorqConfig(templateName string) error {
	return client.DeleteWebhookConfig(templateName)
}
