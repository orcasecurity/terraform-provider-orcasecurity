package api_client

import (
	"fmt"
	"net/url"
)

const externalServiceConfigPath = "/api/external_service/config"

// ConfigEnvelope is the JSON shape every /api/external_service/config integration shares.
// Per-service files declare their own Config type C and alias the envelope:
//
//	type PagerDutyExternalServiceConfig = ConfigEnvelope[PagerDutyConfig]
//
// The Resource field is only meaningful for sn_incidents templates; it carries `omitempty`
// so omitting it on other services keeps the wire payload identical to the pre-refactor shape.
type ConfigEnvelope[C any] struct {
	ID            string   `json:"id,omitempty"`
	ServiceName   string   `json:"service_name,omitempty"`
	TemplateName  string   `json:"template_name,omitempty"`
	Resource      string   `json:"resource,omitempty"`
	Config        C        `json:"config"`
	IsEnabled     bool     `json:"is_enabled"`
	IsDefault     bool     `json:"is_default"`
	BusinessUnits []string `json:"business_units,omitempty"`
	CreatedAt     string   `json:"created_at,omitempty"`
	UpdatedAt     string   `json:"updated_at,omitempty"`
}

type configSingleResponse[C any] struct {
	Status string            `json:"status"`
	Data   ConfigEnvelope[C] `json:"data"`
}

type configListResponse[C any] struct {
	Status string              `json:"status"`
	Data   []ConfigEnvelope[C] `json:"data"`
}

func configResourceURL(serviceName, templateName string) string {
	return fmt.Sprintf("%s/%s?template=%s", externalServiceConfigPath, serviceName, url.QueryEscape(templateName))
}

func configListURL(serviceName, templateName string) string {
	return fmt.Sprintf("%s?service_name=%s&template_name=%s", externalServiceConfigPath, serviceName, url.QueryEscape(templateName))
}

// ExternalServiceConfigCreateOp builds the create for an envelope-shaped variant. serviceName is
// pinned on the payload, so the POST and the recovery lookup always use the same service.
func ExternalServiceConfigCreateOp[C any](client *APIClient, serviceName string, payload ConfigEnvelope[C]) CreateOp[ConfigEnvelope[C]] {
	payload.ServiceName = serviceName
	return externalServiceConfigCreateOp(client, serviceName, payload.TemplateName, payload,
		func(e *ConfigEnvelope[C]) string { return e.ID })
}

// (org, service_name, template_name) is unique, so a replay would 409 on the first copy.
func externalServiceConfigCreateOp[P any](client *APIClient, serviceName, templateName string, payload any, id func(*P) string) CreateOp[P] {
	return CreateOp[P]{
		What: fmt.Sprintf("%s integration with template_name %q", serviceName, templateName),
		Create: func() (*P, error) {
			resp, err := client.PostCreate(externalServiceConfigPath, payload)
			if err != nil {
				return nil, err
			}
			response := struct {
				Data P `json:"data"`
			}{}
			if err := resp.ReadJSON(&response); err != nil {
				return nil, fmt.Errorf("failed to decode %s create response: %w", serviceName, err)
			}
			if id(&response.Data) == "" {
				return nil, fmt.Errorf("%s integration was not returned by the API", serviceName)
			}
			return &response.Data, nil
		},
		IDs: func() (map[string]bool, error) {
			return ListExternalServiceConfigIDs(client, serviceName, templateName)
		},
		Get: func(id string) (*P, error) {
			matches, err := listExternalServiceConfigs[P](client, serviceName, templateName)
			if err != nil {
				return nil, err
			}
			return matches[id], nil
		},
	}
}

func ListExternalServiceConfigIDs(client *APIClient, serviceName, templateName string) (map[string]bool, error) {
	matches, err := listExternalServiceConfigs[struct{}](client, serviceName, templateName)
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for id := range matches {
		ids[id] = true
	}
	return ids, nil
}

// The list hides configs whose business_units are outside the caller's permitted BUs.
func listExternalServiceConfigs[P any](client *APIClient, serviceName, templateName string) (map[string]*P, error) {
	resp, err := client.Get(configListURL(serviceName, templateName))
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return map[string]*P{}, nil
		}
		return nil, err
	}
	keys := configListResponse[struct{}]{}
	rows := struct {
		Data []P `json:"data"`
	}{}
	if err := resp.ReadJSON(&keys); err != nil {
		return nil, fmt.Errorf("failed to decode %s list response: %w", serviceName, err)
	}
	if err := resp.ReadJSON(&rows); err != nil {
		return nil, fmt.Errorf("failed to decode %s list response: %w", serviceName, err)
	}
	matches := map[string]*P{}
	for i, key := range keys.Data {
		if key.ServiceName == serviceName && sameName(key.TemplateName, templateName) {
			matches[key.ID] = &rows.Data[i]
		}
	}
	return matches, nil
}

// GetExternalServiceConfig fetches the config list for templateName and returns the first
// entry the optional filter accepts. A nil filter returns the first entry. Returns (nil, nil)
// when no entry matches — callers treat that as a deleted-out-of-band signal.
func GetExternalServiceConfig[C any](client *APIClient, serviceName, templateName string, filter func(*ConfigEnvelope[C]) bool) (*ConfigEnvelope[C], error) {
	resp, err := client.Get(configListURL(serviceName, templateName))
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}
	response := configListResponse[C]{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode %s list response: %w", serviceName, err)
	}
	for i := range response.Data {
		if filter == nil || filter(&response.Data[i]) {
			return &response.Data[i], nil
		}
	}
	return nil, nil
}

// UpdateExternalServiceConfig PUTs a partial body. Callers compose the body via BuildUpdateBody
// so each integration controls which secret fields it forwards (empty secrets are omitted so
// the Orca API keeps the value already in SSM).
func UpdateExternalServiceConfig[C any](client *APIClient, serviceName, templateName string, body map[string]interface{}) (*ConfigEnvelope[C], error) {
	resp, err := client.Put(configResourceURL(serviceName, templateName), body)
	if err != nil {
		return nil, err
	}
	response := configSingleResponse[C]{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode %s update response: %w", serviceName, err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("%s integration was not returned by the API", serviceName)
	}
	return &response.Data, nil
}

// DeleteExternalServiceConfig issues a DELETE for the config and discards the response body.
func DeleteExternalServiceConfig(client *APIClient, serviceName, templateName string) error {
	_, err := client.Delete(configResourceURL(serviceName, templateName))
	return err
}

// BuildUpdateBody composes the standard PUT body: is_enabled, is_default, the supplied config
// (struct or map), and — when forwardBusinessUnits is true — business_units. Only integrations
// whose Orca service accepts BU changes on PUT (currently azure_sentinel, opsgenie) set the
// flag. Services that treat business_units as RequiresReplace must leave it false; the field
// would otherwise be silently dropped on the server side with no compile error to catch it.
func BuildUpdateBody[C any](payload ConfigEnvelope[C], cfg interface{}, forwardBusinessUnits bool) map[string]interface{} {
	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
		"config":     cfg,
	}
	if forwardBusinessUnits && payload.BusinessUnits != nil {
		body["business_units"] = payload.BusinessUnits
	}
	return body
}
