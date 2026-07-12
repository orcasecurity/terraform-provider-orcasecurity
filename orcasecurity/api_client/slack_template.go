package api_client

import (
	"encoding/json"
)

const SlackServiceName = "slack"

// SlackConfig mirrors the "config" block of the slack external_service/config payload.
// Mapping is kept as json.RawMessage so the provider preserves the customer's arbitrary
// title/description field structure verbatim (Orca validates server-side).
type SlackConfig struct {
	WorkspaceID string   `json:"workspace_id"`
	Channels    []string `json:"channels"`
	// ShowActions is a pointer so the provider can tell "absent from the API config"
	// apart from an explicit false. Orca omits show_actions from stored config in some
	// cases (e.g. templates created in the UI); a missing value defaults to true, matching
	// the UI (`config?.show_actions ?? true`). A plain bool would decode absent as false and
	// produce a perpetual show_actions diff on import.
	ShowActions *bool           `json:"show_actions,omitempty"`
	Mapping     json.RawMessage `json:"mapping"`
}

// SlackTemplate aliases the shared envelope. Slack has no linked OAuth resource, so the
// envelope's Resource field is left empty.
type SlackTemplate = ConfigEnvelope[SlackConfig]

func (client *APIClient) CreateSlackTemplate(payload SlackTemplate) (*SlackTemplate, error) {
	return CreateExternalServiceConfig[SlackConfig](client, SlackServiceName, payload)
}

func (client *APIClient) GetSlackTemplate(templateName string) (*SlackTemplate, error) {
	return GetExternalServiceConfig[SlackConfig](client, SlackServiceName, templateName, nil)
}

func (client *APIClient) UpdateSlackTemplate(templateName string, payload SlackTemplate) (*SlackTemplate, error) {
	// business_units intentionally omitted — Orca rejects BU changes on update for slack;
	// modelled as RequiresReplace on the Terraform side.
	body := BuildUpdateBody(payload, payload.Config, false)
	return UpdateExternalServiceConfig[SlackConfig](client, SlackServiceName, templateName, body)
}

func (client *APIClient) DeleteSlackTemplate(templateName string) error {
	return DeleteExternalServiceConfig(client, SlackServiceName, templateName)
}
