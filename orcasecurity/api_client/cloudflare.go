package api_client

const CloudflareServiceName = "cloudflare"

type CloudflareConfig struct {
	APIToken string `json:"api_token,omitempty"`
}

type CloudflareExternalServiceConfig = ConfigEnvelope[CloudflareConfig]

func (client *APIClient) CreateCloudflareConfig(payload CloudflareExternalServiceConfig) (*CloudflareExternalServiceConfig, error) {
	return CreateExternalServiceConfig[CloudflareConfig](client, CloudflareServiceName, payload)
}

func (client *APIClient) GetCloudflareConfig(templateName string) (*CloudflareExternalServiceConfig, error) {
	return GetExternalServiceConfig[CloudflareConfig](client, CloudflareServiceName, templateName, nil)
}

func (client *APIClient) UpdateCloudflareConfig(templateName string, payload CloudflareExternalServiceConfig) (*CloudflareExternalServiceConfig, error) {
	cfg := map[string]interface{}{}
	if payload.Config.APIToken != "" {
		cfg["api_token"] = payload.Config.APIToken
	}
	return UpdateExternalServiceConfig[CloudflareConfig](client, CloudflareServiceName, templateName, BuildUpdateBody(payload, cfg, false))
}

func (client *APIClient) DeleteCloudflareConfig(templateName string) error {
	return DeleteExternalServiceConfig(client, CloudflareServiceName, templateName)
}
