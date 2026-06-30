package api_client

const PagerDutyServiceName = "pagerduty"

type PagerDutyConfig struct {
	IntegrationKey string `json:"integration_key,omitempty"`
}

// PagerDutyExternalServiceConfig aliases the generic envelope so existing callers keep their
// type name and field access while CRUD plumbing flows through the shared helper.
type PagerDutyExternalServiceConfig = ConfigEnvelope[PagerDutyConfig]

func (client *APIClient) CreatePagerDutyConfig(payload PagerDutyExternalServiceConfig) (*PagerDutyExternalServiceConfig, error) {
	return CreateExternalServiceConfig[PagerDutyConfig](client, PagerDutyServiceName, payload)
}

func (client *APIClient) GetPagerDutyConfig(templateName string) (*PagerDutyExternalServiceConfig, error) {
	return GetExternalServiceConfig[PagerDutyConfig](client, PagerDutyServiceName, templateName, nil)
}

func (client *APIClient) UpdatePagerDutyConfig(templateName string, payload PagerDutyExternalServiceConfig) (*PagerDutyExternalServiceConfig, error) {
	// PUT body is partial. Omit empty integration_key so the API keeps the value already in SSM.
	cfg := map[string]interface{}{}
	if payload.Config.IntegrationKey != "" {
		cfg["integration_key"] = payload.Config.IntegrationKey
	}
	return UpdateExternalServiceConfig[PagerDutyConfig](client, PagerDutyServiceName, templateName, BuildUpdateBody(payload, cfg, false))
}

func (client *APIClient) DeletePagerDutyConfig(templateName string) error {
	return DeleteExternalServiceConfig(client, PagerDutyServiceName, templateName)
}
