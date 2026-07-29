package api_client

import (
	"encoding/json"
	"fmt"
)

const scheduledReportAPIPath = "/api/reporting/scheduled_reports"

// Nil map marshals as {} — reporting API returns 500 on null object fields.
type JSONObject map[string]interface{}

func (o JSONObject) MarshalJSON() ([]byte, error) {
	if o == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]interface{}(o))
}

type scheduledReportAPIResponseType struct {
	Data ScheduledReport `json:"data"`
}

// ScheduledReport status values as returned by the API.
const (
	ScheduledReportStatusCreated  = 0
	ScheduledReportStatusActive   = 1
	ScheduledReportStatusDisabled = 2
	ScheduledReportStatusArchived = 5
)

type ScheduledReport struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Format     string `json:"format"`
	Recurrence string `json:"recurrence"`

	FirstReportDate string `json:"first_report_date"`
	ExportTime      string `json:"export_time"`

	// Status is an integer enum on the API side (responses always return integers).
	Status *int `json:"status,omitempty"`

	// No omitempty: PATCH omits keys → unchanged. Zero values clear ("" / {}); only id and status keep omitempty.
	Columns          []string   `json:"columns"`
	DSLFilter        JSONObject `json:"dsl_filter"`
	SonarQuery       string     `json:"sonar_query"`
	SonarQueryParams JSONObject `json:"sonar_query_params"`
	QueryFilters     JSONObject `json:"query_filters"`
	Config           JSONObject `json:"config"`
	S3Path           string     `json:"s3_path"`

	RecipientsEmails   []string `json:"recipients_emails"`
	CustomEmailSubject string   `json:"custom_email_subject"`
	CustomEmailContent string   `json:"custom_email_content"`

	ShareToSlack bool       `json:"share_to_slack"`
	SlackChannel JSONObject `json:"slack_channel"`

	ShareToBucket bool   `json:"share_to_bucket"`
	Bucket        string `json:"bucket"`

	ShareToAzureBlob   bool   `json:"share_to_azure_blob"`
	AzureBlobContainer string `json:"azure_blob_container"`

	ShareToGoogleCloudStorage  bool   `json:"share_to_google_cloud_storage"`
	GoogleCloudStorageTemplate string `json:"google_cloud_storage_template"`

	ShareToSnowflake  bool   `json:"share_to_snowflake"`
	SnowflakeTemplate string `json:"snowflake_template"`
}

func (client *APIClient) DoesScheduledReportExist(id string) (bool, error) {
	resp, err := client.Get(fmt.Sprintf("%s/%s", scheduledReportAPIPath, id))
	if resp != nil && (resp.StatusCode() == 404 || resp.StatusCode() == 400) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return resp.IsOk(), nil
}

func (client *APIClient) GetScheduledReport(id string) (*ScheduledReport, error) {
	resp, err := client.Get(fmt.Sprintf("%s/%s", scheduledReportAPIPath, id))
	if resp != nil && (resp.StatusCode() == 404 || resp.StatusCode() == 400) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := scheduledReportAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	report := response.Data
	report.ID = id
	return &report, nil
}

func (client *APIClient) CreateScheduledReport(data ScheduledReport) (*ScheduledReport, error) {
	resp, err := client.Post(scheduledReportAPIPath, data)
	if err != nil {
		return nil, err
	}

	response := scheduledReportAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateScheduledReport(id string, data ScheduledReport) (*ScheduledReport, error) {
	resp, err := client.Patch(fmt.Sprintf("%s/%s", scheduledReportAPIPath, id), data)
	if err != nil {
		return nil, err
	}

	response := scheduledReportAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteScheduledReport(id string) error {
	resp, err := client.Delete(fmt.Sprintf("%s/%s", scheduledReportAPIPath, id))
	// already gone on the remote side
	if resp != nil && resp.StatusCode() == 404 {
		return nil
	}
	return err
}
