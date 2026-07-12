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
	payload.Config.Type = ServiceNowSIRTemplateConfigType
	return CreateExternalServiceConfig[ServiceNowITSMTemplateConfig](client, ServiceNowITSMServiceName, payload)
}

// GetServiceNowSIRTemplate looks up a template by name and filters on “config.type == "SIR"“
// so it does not collide with an ITSM template that happens to share a template_name.
func (client *APIClient) GetServiceNowSIRTemplate(templateName string) (*ServiceNowITSMTemplate, error) {
	return GetExternalServiceConfig[ServiceNowITSMTemplateConfig](client, ServiceNowITSMServiceName, templateName, func(item *ServiceNowITSMTemplate) bool {
		return item.Config.Type == ServiceNowSIRTemplateConfigType
	})
}

func (client *APIClient) UpdateServiceNowSIRTemplate(templateName string, payload ServiceNowITSMTemplate) (*ServiceNowITSMTemplate, error) {
	payload.Config.Type = ServiceNowSIRTemplateConfigType
	// ``business_units`` is intentionally omitted from PUT — Orca rejects updates with
	// "You can't change business units". Matches the ITSM template behaviour.
	body := BuildUpdateBody(payload, payload.Config, false)
	if payload.Resource != "" {
		body["resource"] = payload.Resource
	}
	return UpdateExternalServiceConfig[ServiceNowITSMTemplateConfig](client, ServiceNowITSMServiceName, templateName, body)
}

func (client *APIClient) DeleteServiceNowSIRTemplate(templateName string) error {
	return DeleteExternalServiceConfig(client, ServiceNowITSMServiceName, templateName)
}

// ServiceNowSchemaField is a single field exposed by Orca's ServiceNow schema endpoint.
// Customers use this to discover which ServiceNow fields they can include in “mapping_json“
// on the SIR/ITSM template resources. SIR and ITSM resolve to different ServiceNow tables
// (sn_si_incident vs incident) and therefore return different field sets.
type ServiceNowSchemaField struct {
	Element      string `json:"element"`
	Label        string `json:"label"`
	Type         string `json:"type"`
	DefaultValue string `json:"default_value"`
	MaxLength    string `json:"max_length"`
	Choice       string `json:"choice"`
}

type serviceNowSchemaResponse struct {
	Status string                  `json:"status"`
	Data   []ServiceNowSchemaField `json:"data"`
}

// GetServiceNowSchema lists every mappable field on a ServiceNow table for a given resource.
// snType selects the table variant ("sir" → sn_si_incident, "itsm" → incident). SIR and ITSM
// share the same credentials resource but map to different tables, so their field sets differ.
// Mirrors GET /api/resources/{resource_id}/service_now/{sn_type}/schema.
func (client *APIClient) GetServiceNowSchema(resourceID, snType string) ([]ServiceNowSchemaField, error) {
	resp, err := client.Get(fmt.Sprintf("/api/resources/%s/service_now/%s/schema", url.PathEscape(resourceID), snType))
	if err != nil {
		return nil, err
	}
	response := serviceNowSchemaResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode ServiceNow schema response: %w", err)
	}
	return response.Data, nil
}
