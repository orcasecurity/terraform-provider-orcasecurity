package api_client

const OpsgenieServiceName = "opsgenie"

type OpsgenieConfig struct {
	OpsgenieKey string `json:"opsgenie_key,omitempty"`
}

type OpsgenieExternalServiceConfig = ConfigEnvelope[OpsgenieConfig]

func (client *APIClient) CreateOpsgenieConfig(payload OpsgenieExternalServiceConfig) (*OpsgenieExternalServiceConfig, error) {
	return CreateExternalServiceConfig[OpsgenieConfig](client, OpsgenieServiceName, payload)
}

func (client *APIClient) GetOpsgenieConfig(templateName string) (*OpsgenieExternalServiceConfig, error) {
	return GetExternalServiceConfig[OpsgenieConfig](client, OpsgenieServiceName, templateName, nil)
}

func (client *APIClient) UpdateOpsgenieConfig(templateName string, payload OpsgenieExternalServiceConfig) (*OpsgenieExternalServiceConfig, error) {
	// PUT body is partial. Omit empty opsgenie_key so the API keeps the value already in SSM.
	cfg := map[string]interface{}{}
	if payload.Config.OpsgenieKey != "" {
		cfg["opsgenie_key"] = payload.Config.OpsgenieKey
	}
	return UpdateExternalServiceConfig[OpsgenieConfig](client, OpsgenieServiceName, templateName, BuildUpdateBody(payload, cfg, true))
}

func (client *APIClient) DeleteOpsgenieConfig(templateName string) error {
	return DeleteExternalServiceConfig(client, OpsgenieServiceName, templateName)
}
