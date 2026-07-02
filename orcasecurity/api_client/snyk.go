package api_client

const SnykServiceName = "snyk"

type SnykConfig struct {
	APIToken string `json:"api_token,omitempty"`
	Region   string `json:"region,omitempty"`
}

type SnykExternalServiceConfig = ConfigEnvelope[SnykConfig]

func (client *APIClient) CreateSnykConfig(payload SnykExternalServiceConfig) (*SnykExternalServiceConfig, error) {
	return CreateExternalServiceConfig[SnykConfig](client, SnykServiceName, payload)
}

func (client *APIClient) GetSnykConfig(templateName string) (*SnykExternalServiceConfig, error) {
	return GetExternalServiceConfig[SnykConfig](client, SnykServiceName, templateName, nil)
}

func (client *APIClient) UpdateSnykConfig(templateName string, payload SnykExternalServiceConfig) (*SnykExternalServiceConfig, error) {
	cfg := map[string]interface{}{}
	if payload.Config.APIToken != "" {
		cfg["api_token"] = payload.Config.APIToken
	}
	if payload.Config.Region != "" {
		cfg["region"] = payload.Config.Region
	}
	return UpdateExternalServiceConfig[SnykConfig](client, SnykServiceName, templateName, BuildUpdateBody(payload, cfg, false))
}

func (client *APIClient) DeleteSnykConfig(templateName string) error {
	return DeleteExternalServiceConfig(client, SnykServiceName, templateName)
}
