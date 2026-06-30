package api_client

import (
	"fmt"
	"net/url"
)

const ZscalerServiceName = "zscaler"

type ZscalerConfig struct {
	VanityDomain string `json:"vanity_domain,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

type ZscalerExternalServiceConfig struct {
	ID           string        `json:"id,omitempty"`
	ServiceName  string        `json:"service_name,omitempty"`
	TemplateName string        `json:"template_name,omitempty"`
	Config       ZscalerConfig `json:"config"`
	IsEnabled    bool          `json:"is_enabled"`
	IsDefault    bool          `json:"is_default"`
	CreatedAt    string        `json:"created_at,omitempty"`
	UpdatedAt    string        `json:"updated_at,omitempty"`
}

type zscalerSingleResponse struct {
	Status string                       `json:"status"`
	Data   ZscalerExternalServiceConfig `json:"data"`
}

type zscalerListResponse struct {
	Status string                         `json:"status"`
	Data   []ZscalerExternalServiceConfig `json:"data"`
}

func (client *APIClient) CreateZscalerConfig(payload ZscalerExternalServiceConfig) (*ZscalerExternalServiceConfig, error) {
	payload.ServiceName = ZscalerServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := zscalerSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Zscaler create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("zscaler integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetZscalerConfig(templateName string) (*ZscalerExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		ZscalerServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := zscalerListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Zscaler list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdateZscalerConfig(templateName string, payload ZscalerExternalServiceConfig) (*ZscalerExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		ZscalerServiceName, url.QueryEscape(templateName),
	)

	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
	}
	cfg := map[string]interface{}{}
	if payload.Config.VanityDomain != "" {
		cfg["vanity_domain"] = payload.Config.VanityDomain
	}
	if payload.Config.ClientID != "" {
		cfg["client_id"] = payload.Config.ClientID
	}
	if payload.Config.ClientSecret != "" {
		cfg["client_secret"] = payload.Config.ClientSecret
	}
	body["config"] = cfg

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := zscalerSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Zscaler update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("zscaler integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteZscalerConfig(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		ZscalerServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
