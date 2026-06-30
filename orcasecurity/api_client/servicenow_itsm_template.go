package api_client

import (
	"encoding/json"
	"fmt"
	"net/url"
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

type ServiceNowITSMTemplate struct {
	ID            string                       `json:"id,omitempty"`
	ServiceName   string                       `json:"service_name,omitempty"`
	TemplateName  string                       `json:"template_name,omitempty"`
	Resource      string                       `json:"resource,omitempty"`
	Config        ServiceNowITSMTemplateConfig `json:"config"`
	IsEnabled     bool                         `json:"is_enabled"`
	IsDefault     bool                         `json:"is_default"`
	BusinessUnits []string                     `json:"business_units,omitempty"`
	CreatedAt     string                       `json:"created_at,omitempty"`
	UpdatedAt     string                       `json:"updated_at,omitempty"`
}

type serviceNowITSMTemplateSingleResponse struct {
	Status string                 `json:"status"`
	Data   ServiceNowITSMTemplate `json:"data"`
}

type serviceNowITSMTemplateListResponse struct {
	Status string                   `json:"status"`
	Data   []ServiceNowITSMTemplate `json:"data"`
}

func (client *APIClient) CreateServiceNowITSMTemplate(payload ServiceNowITSMTemplate) (*ServiceNowITSMTemplate, error) {
	payload.ServiceName = ServiceNowITSMServiceName // "sn_incidents"
	payload.Config.Type = "ITSM"

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := serviceNowITSMTemplateSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode ServiceNow ITSM template create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("servicenow itsm template was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetServiceNowITSMTemplate(templateName string) (*ServiceNowITSMTemplate, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		ServiceNowITSMServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := serviceNowITSMTemplateListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode ServiceNow ITSM template list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	// Filter by ITSM type — sn_incidents is shared with SIR.
	for _, item := range response.Data {
		if item.Config.Type == "" || item.Config.Type == "ITSM" {
			return &item, nil
		}
	}
	return nil, nil
}

func (client *APIClient) UpdateServiceNowITSMTemplate(templateName string, payload ServiceNowITSMTemplate) (*ServiceNowITSMTemplate, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		ServiceNowITSMServiceName, url.QueryEscape(templateName),
	)

	payload.Config.Type = "ITSM"
	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
		"config":     payload.Config,
	}
	if payload.Resource != "" {
		body["resource"] = payload.Resource
	}
	// Intentionally omit ``business_units`` from PUT bodies — Orca's external_service/config
	// endpoint rejects updates with ``"You can't change business units"``. Changing the set is
	// modelled as RequiresReplace on the Terraform side.

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := serviceNowITSMTemplateSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode ServiceNow ITSM template update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("servicenow itsm template was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteServiceNowITSMTemplate(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		ServiceNowITSMServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
