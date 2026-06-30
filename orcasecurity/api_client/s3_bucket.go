package api_client

import (
	"fmt"
	"net/url"
)

const S3BucketServiceName = "s3_bucket"

type S3BucketConfig struct {
	ArnOrURL string `json:"arn_or_url,omitempty"`
	Folder   string `json:"folder,omitempty"`
}

type S3BucketExternalServiceConfig struct {
	ID           string         `json:"id,omitempty"`
	ServiceName  string         `json:"service_name,omitempty"`
	TemplateName string         `json:"template_name,omitempty"`
	Config       S3BucketConfig `json:"config"`
	IsEnabled    bool           `json:"is_enabled"`
	IsDefault    bool           `json:"is_default"`
	CreatedAt    string         `json:"created_at,omitempty"`
	UpdatedAt    string         `json:"updated_at,omitempty"`
}

type s3BucketSingleResponse struct {
	Status string                        `json:"status"`
	Data   S3BucketExternalServiceConfig `json:"data"`
}

type s3BucketListResponse struct {
	Status string                          `json:"status"`
	Data   []S3BucketExternalServiceConfig `json:"data"`
}

// OrcaSettings is the public app-settings document exposed by Orca. The provider uses it to
// surface the report_uploader_arn that customers paste into their bucket policy.
type OrcaSettings struct {
	AWSAccountID                             string `json:"aws_account_id"`
	IntegrationCloudformationTemplatesFolder string `json:"integration_cloudformation_templates_folder"`
	ReportUploaderArn                        string `json:"report_uploader_arn"`
	ResourcePartition                        string `json:"resource_partition"`
}

type orcaSettingsResponse struct {
	Status string       `json:"status"`
	Data   OrcaSettings `json:"data"`
}

func (client *APIClient) GetOrcaSettings() (*OrcaSettings, error) {
	resp, err := client.Get("/api/settings")
	if err != nil {
		return nil, err
	}

	// The settings endpoint historically returns either the wrapped {status,data} envelope or
	// the raw settings object — try the envelope first, fall back to the direct shape.
	wrapped := orcaSettingsResponse{}
	if err := resp.ReadJSON(&wrapped); err == nil && wrapped.Data.ReportUploaderArn != "" {
		return &wrapped.Data, nil
	}
	direct := OrcaSettings{}
	if err := resp.ReadJSON(&direct); err == nil && direct.ReportUploaderArn != "" {
		return &direct, nil
	}
	return nil, fmt.Errorf("orca settings response did not include report_uploader_arn")
}

func (client *APIClient) CreateS3BucketConfig(payload S3BucketExternalServiceConfig) (*S3BucketExternalServiceConfig, error) {
	payload.ServiceName = S3BucketServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := s3BucketSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode S3 bucket create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("s3 bucket integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetS3BucketConfig(templateName string) (*S3BucketExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		S3BucketServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := s3BucketListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode S3 bucket list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdateS3BucketConfig(templateName string, payload S3BucketExternalServiceConfig) (*S3BucketExternalServiceConfig, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		S3BucketServiceName, url.QueryEscape(templateName),
	)

	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
	}
	cfg := map[string]interface{}{}
	if payload.Config.ArnOrURL != "" {
		cfg["arn_or_url"] = payload.Config.ArnOrURL
	}
	if payload.Config.Folder != "" {
		cfg["folder"] = payload.Config.Folder
	}
	body["config"] = cfg

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := s3BucketSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode S3 bucket update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("s3 bucket integration was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteS3BucketConfig(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		S3BucketServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
