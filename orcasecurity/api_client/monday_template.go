package api_client

import (
	"encoding/json"
)

const MondayServiceName = "monday"

// MondayTemplateConfig mirrors the "config" block of the monday external_service/config
// payload. The mapping fields are kept as json.RawMessage so the provider preserves the
// customer's arbitrary structure verbatim (Orca validates server-side).
type MondayTemplateConfig struct {
	WorkspaceID         string          `json:"workspace_id"`
	BoardID             string          `json:"board_id,omitempty"`
	GroupID             string          `json:"group_id,omitempty"`
	Mapping             json.RawMessage `json:"mapping"`
	AlertStatusMapping  json.RawMessage `json:"alert_status_mapping,omitempty"`
	TicketStatusMapping json.RawMessage `json:"ticket_status_mapping,omitempty"`
}

// MondayTemplate aliases the shared envelope. The envelope's Resource field carries the linked
// Monday OAuth resource id.
type MondayTemplate = ConfigEnvelope[MondayTemplateConfig]

func (client *APIClient) CreateMondayTemplate(payload MondayTemplate) (*MondayTemplate, error) {
	return CreateExternalServiceConfig[MondayTemplateConfig](client, MondayServiceName, payload)
}

func (client *APIClient) GetMondayTemplate(templateName string) (*MondayTemplate, error) {
	return GetExternalServiceConfig[MondayTemplateConfig](client, MondayServiceName, templateName, nil)
}

func (client *APIClient) UpdateMondayTemplate(templateName string, payload MondayTemplate) (*MondayTemplate, error) {
	// business_units intentionally omitted — Orca rejects BU changes on update
	// ("You can't change business units"); modelled as RequiresReplace on the Terraform side.
	body := BuildUpdateBody(payload, payload.Config, false)
	if payload.Resource != "" {
		body["resource"] = payload.Resource
	}
	return UpdateExternalServiceConfig[MondayTemplateConfig](client, MondayServiceName, templateName, body)
}

func (client *APIClient) DeleteMondayTemplate(templateName string) error {
	return DeleteExternalServiceConfig(client, MondayServiceName, templateName)
}
