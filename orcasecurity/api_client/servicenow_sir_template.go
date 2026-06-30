package api_client

import (
	"fmt"
	"net/url"
)

// SIR templates share the same external_service/config endpoint and JSON shape as ITSM
// templates — only ``config.type`` differs ("SIR" vs "ITSM"). The wrappers below reuse the
// ITSM template payload struct and pin the variant so the SIR resource doesn't have to
// duplicate the model.

const ServiceNowSIRTemplateConfigType = "SIR"

func (client *APIClient) CreateServiceNowSIRTemplate(payload ServiceNowITSMTemplate) (*ServiceNowITSMTemplate, error) {
	payload.ServiceName = ServiceNowITSMServiceName
	payload.Config.Type = ServiceNowSIRTemplateConfigType

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := serviceNowITSMTemplateSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode ServiceNow SIR template create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("servicenow sir template was not returned by the API")
	}
	return &response.Data, nil
}

// GetServiceNowSIRTemplate looks up a template by name and filters on “config.type == "SIR"“
// so it does not collide with an ITSM template that happens to share a template_name.
func (client *APIClient) GetServiceNowSIRTemplate(templateName string) (*ServiceNowITSMTemplate, error) {
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
		return nil, fmt.Errorf("failed to decode ServiceNow SIR template list response: %w", err)
	}
	for _, item := range response.Data {
		if item.Config.Type == ServiceNowSIRTemplateConfigType {
			return &item, nil
		}
	}
	return nil, nil
}

func (client *APIClient) UpdateServiceNowSIRTemplate(templateName string, payload ServiceNowITSMTemplate) (*ServiceNowITSMTemplate, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		ServiceNowITSMServiceName, url.QueryEscape(templateName),
	)

	payload.Config.Type = ServiceNowSIRTemplateConfigType
	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
		"config":     payload.Config,
	}
	if payload.Resource != "" {
		body["resource"] = payload.Resource
	}
	// ``business_units`` is intentionally omitted from PUT — Orca rejects updates with
	// "You can't change business units". Matches the ITSM template behaviour.

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := serviceNowITSMTemplateSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode ServiceNow SIR template update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("servicenow sir template was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteServiceNowSIRTemplate(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		ServiceNowITSMServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}

// ServiceNowSIRSchemaField is a single field exposed by Orca's SIR schema endpoint. Customers
// use this to discover which ServiceNow fields they can include in “mapping_json“ on the
// SIR template resource.
type ServiceNowSIRSchemaField struct {
	Element      string `json:"element"`
	Label        string `json:"label"`
	Type         string `json:"type"`
	DefaultValue string `json:"default_value"`
	MaxLength    string `json:"max_length"`
	Choice       string `json:"choice"`
}

type serviceNowSIRSchemaResponse struct {
	Status string                     `json:"status"`
	Data   []ServiceNowSIRSchemaField `json:"data"`
}

// GetServiceNowSIRSchema lists every mappable field on the ServiceNow SIR table for a given
// resource. Mirrors GET /api/resources/{resource_id}/service_now/sir/schema.
func (client *APIClient) GetServiceNowSIRSchema(resourceID string) ([]ServiceNowSIRSchemaField, error) {
	resp, err := client.Get(fmt.Sprintf("/api/resources/%s/service_now/sir/schema", url.PathEscape(resourceID)))
	if err != nil {
		return nil, err
	}
	response := serviceNowSIRSchemaResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode ServiceNow SIR schema response: %w", err)
	}
	return response.Data, nil
}
