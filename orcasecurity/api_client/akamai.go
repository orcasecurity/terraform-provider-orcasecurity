package api_client

import (
	"fmt"
	"net/url"
)

const AkamaiServiceName = "akamai"

type AkamaiConfig struct {
	AccessToken  string `json:"access_token,omitempty"`
	ClientToken  string `json:"client_token,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Host         string `json:"host,omitempty"`
}

type AkamaiExternalServiceConfig struct {
	ID           string       `json:"id,omitempty"`
	ServiceName  string       `json:"service_name,omitempty"`
	TemplateName string       `json:"template_name,omitempty"`
	Config       AkamaiConfig `json:"config"`
	IsEnabled    bool         `json:"is_enabled"`
	IsDefault    bool         `json:"is_default"`
	CreatedAt    string       `json:"created_at,omitempty"`
	UpdatedAt    string       `json:"updated_at,omitempty"`
}

type akamaiSingleResponse struct {
	Status string                      `json:"status"`
	Data   AkamaiExternalServiceConfig `json:"data"`
}

type akamaiListResponse struct {
	Status string                        `json:"status"`
	Data   []AkamaiExternalServiceConfig `json:"data"`
}

func (client *APIClient) CreateAkamaiConfig(payload AkamaiExternalServiceConfig) (*AkamaiExternalServiceConfig, error) {
	payload.ServiceName = AkamaiServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := akamaiSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Akamai create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Akamai integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetAkamaiConfig(templateName string) (*AkamaiExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		AkamaiServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := akamaiListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Akamai list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdateAkamaiConfig(templateName string, payload AkamaiExternalServiceConfig) (*AkamaiExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		AkamaiServiceName, url.QueryEscape(templateName),
	)

	// PUT body is partial. Omit empty secret fields so the API keeps the value already in SSM
	// when the user did not change them.
	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
	}
	cfg := map[string]interface{}{}
	if payload.Config.AccessToken != "" {
		cfg["access_token"] = payload.Config.AccessToken
	}
	if payload.Config.ClientToken != "" {
		cfg["client_token"] = payload.Config.ClientToken
	}
	if payload.Config.ClientSecret != "" {
		cfg["client_secret"] = payload.Config.ClientSecret
	}
	if payload.Config.Host != "" {
		cfg["host"] = payload.Config.Host
	}
	body["config"] = cfg

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := akamaiSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Akamai update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Akamai integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteAkamaiConfig(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		AkamaiServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
