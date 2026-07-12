package api_client

import (
	"encoding/json"
)

// ServiceNowITSMTemplateConfig mirrors the sn_incidents external_service config payload for
// ITSM-mode templates. The mapping and on_close_alert_mapping fields are kept as
// json.RawMessage so the provider can preserve the customer's arbitrary key/value structure
// (Orca validates them server-side against sn_incidents.schema.json).
type ServiceNowITSMTemplateConfig struct {
	Type                     string          `json:"type"`
	InstanceName             string          `json:"instance_name,omitempty"`
	BaseURL                  string          `json:"base_url,omitempty"`
	Username                 string          `json:"username,omitempty"`
	Password                 string          `json:"password,omitempty"`
	ResolutionStatus         string          `json:"resolution_status,omitempty"`
	ResolutionCode           string          `json:"resolution_code,omitempty"`
	ResolutionNote           string          `json:"resolution_note,omitempty"`
	ReopenStatus             string          `json:"reopen_status,omitempty"`
	Mapping                  json.RawMessage `json:"mapping,omitempty"`
	OnCloseAlertMapping      json.RawMessage `json:"on_close_alert_mapping,omitempty"`
	AllowReopenAndResolution *bool           `json:"allow_reopen_and_resolution,omitempty"`
	AllowMapping             *bool           `json:"allow_mapping,omitempty"`
}

// ServiceNowITSMTemplate aliases the shared envelope. The envelope's Resource field carries
// the linked orcasecurity_integration_servicenow id when set.
type ServiceNowITSMTemplate = ConfigEnvelope[ServiceNowITSMTemplateConfig]

const ServiceNowITSMTemplateConfigType = "ITSM"

func (client *APIClient) CreateServiceNowITSMTemplate(payload ServiceNowITSMTemplate) (*ServiceNowITSMTemplate, error) {
	payload.Config.Type = ServiceNowITSMTemplateConfigType
	return CreateExternalServiceConfig[ServiceNowITSMTemplateConfig](client, ServiceNowITSMServiceName, payload)
}

// GetServiceNowITSMTemplate looks up a template by name and filters on “config.type == "ITSM"“
// (or the legacy unset value) so it does not collide with a SIR template that happens to
// share a template_name.
func (client *APIClient) GetServiceNowITSMTemplate(templateName string) (*ServiceNowITSMTemplate, error) {
	return GetExternalServiceConfig[ServiceNowITSMTemplateConfig](client, ServiceNowITSMServiceName, templateName, func(item *ServiceNowITSMTemplate) bool {
		return item.Config.Type == "" || item.Config.Type == ServiceNowITSMTemplateConfigType
	})
}

func (client *APIClient) UpdateServiceNowITSMTemplate(templateName string, payload ServiceNowITSMTemplate) (*ServiceNowITSMTemplate, error) {
	payload.Config.Type = ServiceNowITSMTemplateConfigType
	// Intentionally omit ``business_units`` from PUT bodies — Orca's external_service/config
	// endpoint rejects updates with ``"You can't change business units"``. Changing the set is
	// modelled as RequiresReplace on the Terraform side.
	body := BuildUpdateBody(payload, payload.Config, false)
	if payload.Resource != "" {
		body["resource"] = payload.Resource
	}
	return UpdateExternalServiceConfig[ServiceNowITSMTemplateConfig](client, ServiceNowITSMServiceName, templateName, body)
}

func (client *APIClient) DeleteServiceNowITSMTemplate(templateName string) error {
	return DeleteExternalServiceConfig(client, ServiceNowITSMServiceName, templateName)
}
