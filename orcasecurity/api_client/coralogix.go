package api_client

// Coralogix is exposed by Orca as a webhook variant: the API stores it under
// service_name="webhook" with config.type="coralogix". The wrappers below pin the variant so
// callers can treat Coralogix as its own integration without leaking the webhook plumbing.

const CoralogixWebhookType = "coralogix"

func (client *APIClient) CreateCoralogixConfig(payload WebhookExternalServiceConfig) (*WebhookExternalServiceConfig, error) {
	payload.Config.Type = CoralogixWebhookType
	return client.CreateWebhookConfig(payload)
}

func (client *APIClient) GetCoralogixConfig(templateName string) (*WebhookExternalServiceConfig, error) {
	return client.GetWebhookConfigByTemplate(templateName)
}

func (client *APIClient) UpdateCoralogixConfig(templateName string, payload WebhookExternalServiceConfig) (*WebhookExternalServiceConfig, error) {
	payload.Config.Type = CoralogixWebhookType
	return client.UpdateWebhookConfig(templateName, payload)
}

func (client *APIClient) DeleteCoralogixConfig(templateName string) error {
	return client.DeleteWebhookConfig(templateName)
}
