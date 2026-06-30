package api_client

const ZscalerServiceName = "zscaler"

type ZscalerConfig struct {
	VanityDomain string `json:"vanity_domain,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

type ZscalerExternalServiceConfig = ConfigEnvelope[ZscalerConfig]

func (client *APIClient) CreateZscalerConfig(payload ZscalerExternalServiceConfig) (*ZscalerExternalServiceConfig, error) {
	return CreateExternalServiceConfig[ZscalerConfig](client, ZscalerServiceName, payload)
}

func (client *APIClient) GetZscalerConfig(templateName string) (*ZscalerExternalServiceConfig, error) {
	return GetExternalServiceConfig[ZscalerConfig](client, ZscalerServiceName, templateName, nil)
}

func (client *APIClient) UpdateZscalerConfig(templateName string, payload ZscalerExternalServiceConfig) (*ZscalerExternalServiceConfig, error) {
	cfg := map[string]interface{}{}
	if payload.Config.VanityDomain != "" {
		cfg["vanity_domain"] = payload.Config.VanityDomain
	}
	if payload.Config.ClientID != "" {
		cfg["client_id"] = payload.Config.ClientID
	}
	if payload.Config.ClientSecret != "" {
		cfg["client_secret"] = payload.Config.ClientSecret
	}
	return UpdateExternalServiceConfig[ZscalerConfig](client, ZscalerServiceName, templateName, BuildUpdateBody(payload, cfg, false))
}

func (client *APIClient) DeleteZscalerConfig(templateName string) error {
	return DeleteExternalServiceConfig(client, ZscalerServiceName, templateName)
}
