package api_client

const AkamaiServiceName = "akamai"

type AkamaiConfig struct {
	AccessToken  string `json:"access_token,omitempty"`
	ClientToken  string `json:"client_token,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Host         string `json:"host,omitempty"`
}

type AkamaiExternalServiceConfig = ConfigEnvelope[AkamaiConfig]

func (client *APIClient) CreateAkamaiConfig(payload AkamaiExternalServiceConfig) (*AkamaiExternalServiceConfig, error) {
	return CreateExternalServiceConfig[AkamaiConfig](client, AkamaiServiceName, payload)
}

func (client *APIClient) GetAkamaiConfig(templateName string) (*AkamaiExternalServiceConfig, error) {
	return GetExternalServiceConfig[AkamaiConfig](client, AkamaiServiceName, templateName, nil)
}

func (client *APIClient) UpdateAkamaiConfig(templateName string, payload AkamaiExternalServiceConfig) (*AkamaiExternalServiceConfig, error) {
	// PUT body is partial. Omit empty secret fields so the API keeps the SSM-resident value
	// when the user did not change them.
	cfg := map[string]interface{}{}
	if payload.Config.AccessToken != "" {
		cfg["access_token"] = payload.Config.AccessToken
	}
	if payload.Config.ClientToken != "" {
		cfg["client_token"] = payload.Config.ClientToken
	}
	if payload.Config.ClientSecret != "" {
		cfg["client_secret"] = payload.Config.ClientSecret
	}
	if payload.Config.Host != "" {
		cfg["host"] = payload.Config.Host
	}
	return UpdateExternalServiceConfig[AkamaiConfig](client, AkamaiServiceName, templateName, BuildUpdateBody(payload, cfg, false))
}

func (client *APIClient) DeleteAkamaiConfig(templateName string) error {
	return DeleteExternalServiceConfig(client, AkamaiServiceName, templateName)
}
