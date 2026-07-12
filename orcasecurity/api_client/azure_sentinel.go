package api_client

const AzureSentinelServiceName = "azure_sentinel"

type AzureSentinelConfig struct {
	LogType     string `json:"log_type,omitempty"`
	PrimaryKey  string `json:"primary_key,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

type AzureSentinelExternalServiceConfig = ConfigEnvelope[AzureSentinelConfig]

func (client *APIClient) CreateAzureSentinelConfig(payload AzureSentinelExternalServiceConfig) (*AzureSentinelExternalServiceConfig, error) {
	return CreateExternalServiceConfig[AzureSentinelConfig](client, AzureSentinelServiceName, payload)
}

func (client *APIClient) GetAzureSentinelConfig(templateName string) (*AzureSentinelExternalServiceConfig, error) {
	return GetExternalServiceConfig[AzureSentinelConfig](client, AzureSentinelServiceName, templateName, nil)
}

func (client *APIClient) UpdateAzureSentinelConfig(templateName string, payload AzureSentinelExternalServiceConfig) (*AzureSentinelExternalServiceConfig, error) {
	cfg := map[string]interface{}{}
	if payload.Config.LogType != "" {
		cfg["log_type"] = payload.Config.LogType
	}
	if payload.Config.PrimaryKey != "" {
		cfg["primary_key"] = payload.Config.PrimaryKey
	}
	if payload.Config.WorkspaceID != "" {
		cfg["workspace_id"] = payload.Config.WorkspaceID
	}
	return UpdateExternalServiceConfig[AzureSentinelConfig](client, AzureSentinelServiceName, templateName, BuildUpdateBody(payload, cfg, true))
}

func (client *APIClient) DeleteAzureSentinelConfig(templateName string) error {
	return DeleteExternalServiceConfig(client, AzureSentinelServiceName, templateName)
}
