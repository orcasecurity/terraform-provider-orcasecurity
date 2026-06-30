package api_client

import (
	"fmt"
	"net/url"
)

const TerraformCloudServiceName = "terraform_cloud"

type TerraformCloudConfig struct {
	APIToken string `json:"api_token,omitempty"`
	APIURL   string `json:"api_url,omitempty"`
}

type TerraformCloudExternalServiceConfig struct {
	ID           string               `json:"id,omitempty"`
	ServiceName  string               `json:"service_name,omitempty"`
	TemplateName string               `json:"template_name,omitempty"`
	Config       TerraformCloudConfig `json:"config"`
	IsEnabled    bool                 `json:"is_enabled"`
	IsDefault    bool                 `json:"is_default"`
	CreatedAt    string               `json:"created_at,omitempty"`
	UpdatedAt    string               `json:"updated_at,omitempty"`
}

type terraformCloudSingleResponse struct {
	Status string                              `json:"status"`
	Data   TerraformCloudExternalServiceConfig `json:"data"`
}

type terraformCloudListResponse struct {
	Status string                                `json:"status"`
	Data   []TerraformCloudExternalServiceConfig `json:"data"`
}

func (client *APIClient) CreateTerraformCloudConfig(payload TerraformCloudExternalServiceConfig) (*TerraformCloudExternalServiceConfig, error) {
	payload.ServiceName = TerraformCloudServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := terraformCloudSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Terraform Cloud create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Terraform Cloud integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetTerraformCloudConfig(templateName string) (*TerraformCloudExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		TerraformCloudServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := terraformCloudListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Terraform Cloud list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdateTerraformCloudConfig(templateName string, payload TerraformCloudExternalServiceConfig) (*TerraformCloudExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		TerraformCloudServiceName, url.QueryEscape(templateName),
	)

	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
	}
	cfg := map[string]interface{}{}
	if payload.Config.APIToken != "" {
		cfg["api_token"] = payload.Config.APIToken
	}
	if payload.Config.APIURL != "" {
		cfg["api_url"] = payload.Config.APIURL
	}
	body["config"] = cfg

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := terraformCloudSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Terraform Cloud update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Terraform Cloud integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteTerraformCloudConfig(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		TerraformCloudServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
