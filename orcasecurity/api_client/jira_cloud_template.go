package api_client

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const JiraCloudServiceName = "jira"

// JiraCloudTemplateConfig mirrors the ``config`` block of the Jira Cloud
// external_service/config payload. The free-form mapping fields are kept as json.RawMessage
// so the provider can preserve the customer's structure verbatim (Orca validates server-side).
type JiraCloudTemplateConfig struct {
	ResourceID                 string          `json:"resource_id,omitempty"`
	ResourceURL                string          `json:"resource_url,omitempty"`
	ProjectID                  string          `json:"project_id,omitempty"`
	IssueTypeID                string          `json:"issue_type_id,omitempty"`
	SubtaskIssueTypeID         string          `json:"subtask_issue_type_id,omitempty"`
	Mapping                    json.RawMessage `json:"mapping,omitempty"`
	AlertStatusMapping         json.RawMessage `json:"alert_status_mapping,omitempty"`
	TicketStatusMapping        json.RawMessage `json:"ticket_status_mapping,omitempty"`
	SubtaskAlertStatusMapping  json.RawMessage `json:"subtask_alert_status_mapping,omitempty"`
	SubtaskTicketStatusMapping json.RawMessage `json:"subtask_ticket_status_mapping,omitempty"`
}

type JiraCloudTemplate struct {
	ID            string                  `json:"id,omitempty"`
	ServiceName   string                  `json:"service_name,omitempty"`
	TemplateName  string                  `json:"template_name,omitempty"`
	Config        JiraCloudTemplateConfig `json:"config"`
	IsEnabled     bool                    `json:"is_enabled"`
	IsDefault     bool                    `json:"is_default"`
	BusinessUnits []string                `json:"business_units,omitempty"`
	CreatedAt     string                  `json:"created_at,omitempty"`
	UpdatedAt     string                  `json:"updated_at,omitempty"`
}

type jiraCloudSingleResponse struct {
	Status string            `json:"status"`
	Data   JiraCloudTemplate `json:"data"`
}

type jiraCloudListResponse struct {
	Status string              `json:"status"`
	Data   []JiraCloudTemplate `json:"data"`
}

func (client *APIClient) CreateJiraCloudTemplate(payload JiraCloudTemplate) (*JiraCloudTemplate, error) {
	payload.ServiceName = JiraCloudServiceName

	resp, err := client.Post("/api/external_service/config", payload)
	if err != nil {
		return nil, err
	}

	response := jiraCloudSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Jira Cloud create response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Jira Cloud template was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) GetJiraCloudTemplate(templateName string) (*JiraCloudTemplate, error) {
	path := fmt.Sprintf(
		"/api/external_service/config?service_name=%s&template_name=%s",
		JiraCloudServiceName, url.QueryEscape(templateName),
	)
	resp, err := client.Get(path)
	if err != nil {
		if resp != nil && resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, err
	}

	response := jiraCloudListResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Jira Cloud list response: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}
	return &response.Data[0], nil
}

func (client *APIClient) UpdateJiraCloudTemplate(templateName string, payload JiraCloudTemplate) (*JiraCloudTemplate, error) {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		JiraCloudServiceName, url.QueryEscape(templateName),
	)

	body := map[string]interface{}{
		"is_enabled": payload.IsEnabled,
		"is_default": payload.IsDefault,
		"config":     payload.Config,
	}
	// business_units intentionally omitted — Orca rejects updates with
	// "You can't change business units". Modelled as RequiresReplace on the Terraform side.

	resp, err := client.Put(path, body)
	if err != nil {
		return nil, err
	}

	response := jiraCloudSingleResponse{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Jira Cloud update response: %w", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("Jira Cloud template was not returned by the API")
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteJiraCloudTemplate(templateName string) error {
	path := fmt.Sprintf(
		"/api/external_service/config/%s?template=%s",
		JiraCloudServiceName, url.QueryEscape(templateName),
	)
	_, err := client.Delete(path)
	return err
}
