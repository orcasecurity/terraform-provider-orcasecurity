package api_client

import (
	"fmt"
	"net/url"
)

const SplunkServiceName = "splunk"

type SplunkConfig struct {
	URL                 string `json:"url,omitempty"`
	Token               string `json:"token,omitempty"`
	AllowSelfSignedCert bool   `json:"allow_self_signed_cert"`
}

type SplunkExternalServiceConfig struct {
	ID           string       `json:"id,omitempty"`
	ServiceName  string       `json:"service_name,omitempty"`
	TemplateName string       `json:"template_name,omitempty"`
	Config       SplunkConfig `json:"config"`
	IsEnabled    bool         `json:"is_enabled"`
	IsDefault    bool         `json:"is_default"`
	CreatedAt    string       `json:"created_at,omitempty"`
	UpdatedAt    string       `json:"updated_at,omitempty"`
}

type splunkSingleResponse struct {
	Status string                      `json:"status"`
	Data   SplunkExternalServiceConfig `json:"data"`
}

type splunkListResponse struct {
	Status string                        `json:"status"`
	Data   []SplunkExternalServiceConfig `json:"data"`
}

func (client *APIClient) CreateSplunkConfig(payload SplunkExternalServiceConfig) (*SplunkExternalServiceConfig, error) {
	payload.ServiceName = SplunkServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := splunkSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Splunk create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Splunk integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetSplunkConfig(templateName string) (*SplunkExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		SplunkServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := splunkListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Splunk list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdateSplunkConfig(templateName string, payload SplunkExternalServiceConfig) (*SplunkExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		SplunkServiceName, url.QueryEscape(templateName),
	)

	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
	}
	cfg := map[string]interface{}{
		"allow_self_signed_cert": payload.Config.AllowSelfSignedCert,
	}
	if payload.Config.URL != "" {
		cfg["url"] = payload.Config.URL
	}
	if payload.Config.Token != "" {
		cfg["token"] = payload.Config.Token
	}
	body["config"] = cfg

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := splunkSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Splunk update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Splunk integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteSplunkConfig(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		SplunkServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
