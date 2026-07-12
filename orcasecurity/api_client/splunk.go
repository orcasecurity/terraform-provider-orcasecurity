package api_client

const SplunkServiceName = "splunk"

type SplunkConfig struct {
	URL                 string `json:"url,omitempty"`
	Token               string `json:"token,omitempty"`
	AllowSelfSignedCert bool   `json:"allow_self_signed_cert"`
}

type SplunkExternalServiceConfig = ConfigEnvelope[SplunkConfig]

func (client *APIClient) CreateSplunkConfig(payload SplunkExternalServiceConfig) (*SplunkExternalServiceConfig, error) {
	return CreateExternalServiceConfig[SplunkConfig](client, SplunkServiceName, payload)
}

func (client *APIClient) GetSplunkConfig(templateName string) (*SplunkExternalServiceConfig, error) {
	return GetExternalServiceConfig[SplunkConfig](client, SplunkServiceName, templateName, nil)
}

func (client *APIClient) UpdateSplunkConfig(templateName string, payload SplunkExternalServiceConfig) (*SplunkExternalServiceConfig, error) {
	cfg := map[string]interface{}{
		"allow_self_signed_cert": payload.Config.AllowSelfSignedCert,
	}
	if payload.Config.URL != "" {
		cfg["url"] = payload.Config.URL
	}
	if payload.Config.Token != "" {
		cfg["token"] = payload.Config.Token
	}
	return UpdateExternalServiceConfig[SplunkConfig](client, SplunkServiceName, templateName, BuildUpdateBody(payload, cfg, false))
}

func (client *APIClient) DeleteSplunkConfig(templateName string) error {
	return DeleteExternalServiceConfig(client, SplunkServiceName, templateName)
}
