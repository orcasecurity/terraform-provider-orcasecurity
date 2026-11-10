package api_client

import (
	"encoding/json"
	"fmt"
	"strings"
)

const scheduledReportAPIPath = "/api/reporting/scheduled_reports"

// Nil map marshals as {} — reporting API returns 500 on null object fields.
type jsonObject map[string]interface{}

func (o jsonObject) MarshalJSON() ([]byte, error) {
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
	// Writes go through scheduledReportBody, which omits unset optionals that hold nothing to clear.
	Columns          []string   `json:"columns"`
	DSLFilter        jsonObject `json:"dsl_filter"`
	SonarQuery       string     `json:"sonar_query"`
	SonarQueryParams jsonObject `json:"sonar_query_params"`
	QueryFilters     jsonObject `json:"query_filters"`
	Config           jsonObject `json:"config"`
	S3Path           string     `json:"s3_path"`

	RecipientsEmails   []string `json:"recipients_emails"`
	CustomEmailSubject string   `json:"custom_email_subject"`
	CustomEmailContent string   `json:"custom_email_content"`

	ShareToSlack bool       `json:"share_to_slack"`
	SlackChannel jsonObject `json:"slack_channel"`

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

var scheduledReportClearableFields = []string{
	"dsl_filter", "sonar_query", "sonar_query_params", "query_filters", "config", "s3_path",
	"custom_email_subject", "custom_email_content", "slack_channel", "bucket",
	"azure_blob_container", "google_cloud_storage_template", "snowflake_template",
}

func scheduledReportFields(data ScheduledReport) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	body := map[string]json.RawMessage{}
	err = json.Unmarshal(raw, &body)
	return body, err
}

func isScheduledReportZero(v json.RawMessage) bool {
	return len(v) == 0 || string(v) == `""` || string(v) == "{}"
}

// The API stores a sent "" as "" but an omitted string as null, and PATCH treats an omitted key
// as unchanged. So an unset optional is sent as its zero value only to clear a value prior held;
// otherwise it is omitted (create has no prior, so it omits every unset optional).
func scheduledReportBody(prior *ScheduledReport, next ScheduledReport) (map[string]json.RawMessage, error) {
	body, err := scheduledReportFields(next)
	if err != nil {
		return nil, err
	}
	held := map[string]json.RawMessage{}
	if prior != nil {
		if held, err = scheduledReportFields(*prior); err != nil {
			return nil, err
		}
	}
	for _, field := range scheduledReportClearableFields {
		if isScheduledReportZero(body[field]) && isScheduledReportZero(held[field]) {
			delete(body, field)
		}
	}
	return body, nil
}

func (client *APIClient) createScheduledReport(data ScheduledReport) (*ScheduledReport, error) {
	body, err := scheduledReportBody(nil, data)
	if err != nil {
		return nil, err
	}
	resp, err := client.PostCreate(scheduledReportAPIPath, body)
	if err != nil {
		return nil, err
	}

	response := scheduledReportAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// The server stores "/" in a name as ":"; search is icontains over name and creator.
func (client *APIClient) ListScheduledReportIDsByName(name string) (map[string]bool, error) {
	stored := strings.ReplaceAll(name, "/", ":")
	all, err := paginateOffset[ScheduledReport](client, scheduledReportAPIPath, listFilters{"search": strings.TrimSpace(stored)}, 300, 500)
	if err != nil {
		return nil, err
	}
	return idsWhere(all, func(r *ScheduledReport) string { return r.ID }, func(r *ScheduledReport) bool { return sameName(r.Name, stored) }), nil
}

func (client *APIClient) ScheduledReportCreateOp(data ScheduledReport) CreateOp[ScheduledReport] {
	return CreateOp[ScheduledReport]{
		What:   fmt.Sprintf("scheduled report named %q", data.Name),
		Create: func() (*ScheduledReport, error) { return client.createScheduledReport(data) },
		IDs:    func() (map[string]bool, error) { return client.ListScheduledReportIDsByName(data.Name) },
		Get:    client.GetScheduledReport,
	}
}

func (client *APIClient) UpdateScheduledReport(id string, prior, next ScheduledReport) (*ScheduledReport, error) {
	body, err := scheduledReportBody(&prior, next)
	if err != nil {
		return nil, err
	}
	resp, err := client.Patch(fmt.Sprintf("%s/%s", scheduledReportAPIPath, id), body)
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
