package api_client

import (
	"fmt"
	"net/url"
)

const CloudflareServiceName = "cloudflare"

type CloudflareConfig struct {
	APIToken string `json:"api_token,omitempty"`
}

type CloudflareExternalServiceConfig struct {
	ID           string           `json:"id,omitempty"`
	ServiceName  string           `json:"service_name,omitempty"`
	TemplateName string           `json:"template_name,omitempty"`
	Config       CloudflareConfig `json:"config"`
	IsEnabled    bool             `json:"is_enabled"`
	IsDefault    bool             `json:"is_default"`
	CreatedAt    string           `json:"created_at,omitempty"`
	UpdatedAt    string           `json:"updated_at,omitempty"`
}

type cloudflareSingleResponse struct {
	Status string                          `json:"status"`
	Data   CloudflareExternalServiceConfig `json:"data"`
}

type cloudflareListResponse struct {
	Status string                            `json:"status"`
	Data   []CloudflareExternalServiceConfig `json:"data"`
}

func (client *APIClient) CreateCloudflareConfig(payload CloudflareExternalServiceConfig) (*CloudflareExternalServiceConfig, error) {
	payload.ServiceName = CloudflareServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := cloudflareSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Cloudflare create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Cloudflare integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetCloudflareConfig(templateName string) (*CloudflareExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		CloudflareServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := cloudflareListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Cloudflare list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdateCloudflareConfig(templateName string, payload CloudflareExternalServiceConfig) (*CloudflareExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		CloudflareServiceName, url.QueryEscape(templateName),
	)

	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
	}
	cfg := map[string]interface{}{}
	if payload.Config.APIToken != "" {
		cfg["api_token"] = payload.Config.APIToken
	}
	body["config"] = cfg

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := cloudflareSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Cloudflare update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Cloudflare integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteCloudflareConfig(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		CloudflareServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
