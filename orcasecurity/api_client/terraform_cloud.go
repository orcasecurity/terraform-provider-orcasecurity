package api_client

const TerraformCloudServiceName = "terraform_cloud"

type TerraformCloudConfig struct {
	APIToken string `json:"api_token,omitempty"`
	APIURL   string `json:"api_url,omitempty"`
}

type TerraformCloudExternalServiceConfig = ConfigEnvelope[TerraformCloudConfig]

func (client *APIClient) CreateTerraformCloudConfig(payload TerraformCloudExternalServiceConfig) (*TerraformCloudExternalServiceConfig, error) {
	return CreateExternalServiceConfig[TerraformCloudConfig](client, TerraformCloudServiceName, payload)
}

func (client *APIClient) GetTerraformCloudConfig(templateName string) (*TerraformCloudExternalServiceConfig, error) {
	return GetExternalServiceConfig[TerraformCloudConfig](client, TerraformCloudServiceName, templateName, nil)
}

func (client *APIClient) UpdateTerraformCloudConfig(templateName string, payload TerraformCloudExternalServiceConfig) (*TerraformCloudExternalServiceConfig, error) {
	cfg := map[string]interface{}{}
	if payload.Config.APIToken != "" {
		cfg["api_token"] = payload.Config.APIToken
	}
	if payload.Config.APIURL != "" {
		cfg["api_url"] = payload.Config.APIURL
	}
	return UpdateExternalServiceConfig[TerraformCloudConfig](client, TerraformCloudServiceName, templateName, BuildUpdateBody(payload, cfg, false))
}

func (client *APIClient) DeleteTerraformCloudConfig(templateName string) error {
	return DeleteExternalServiceConfig(client, TerraformCloudServiceName, templateName)
}
