package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const testScheduledReportID = "2b0fc6d4-4b3a-4f44-86b9-6e80881e2a5e"

const testScheduledReportResponse = `{
	"status": "success",
	"data": {
		"id": "2b0fc6d4-4b3a-4f44-86b9-6e80881e2a5e",
		"name": "Weekly open alerts",
		"type": "alerts_svl",
		"format": "csv",
		"recurrence": "weekly",
		"status": 1,
		"first_report_date": "2026-06-11T13:00:00Z",
		"export_time": "13:00:00",
		"sonar_query": "{\"models\":[\"Alert\"],\"type\":\"object_set\"}",
		"sonar_query_params": {"max_tier": 5},
		"columns": ["OrcaScore", "Title"],
		"recipients_emails": ["test@orca.security"],
		"config": {"compression_type": ".zip"},
		"share_to_slack": false,
		"share_to_bucket": false,
		"share_to_azure_blob": false,
		"share_to_google_cloud_storage": false,
		"share_to_snowflake": false,
		"created_by": {"id": "x", "email": "test@orca.security"},
		"total_generated_reports": 0
	}
}`

func assertScheduledReportCreateRequest(t *testing.T, req *http.Request) {
	t.Helper()

	if req.Method != "POST" {
		t.Errorf("expected POST, got %s", req.Method)
	}
	if !strings.HasSuffix(req.URL.Path, "/api/reporting/scheduled_reports") {
		t.Errorf("unexpected path: %s", req.URL.Path)
	}

	body, _ := io.ReadAll(req.Body)
	payload := map[string]interface{}{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}

	expectedValues := map[string]interface{}{
		"name":   "Weekly open alerts",
		"type":   "alerts_svl",
		"status": float64(1), // status is sent as an integer
	}
	for field, expected := range expectedValues {
		if payload[field] != expected {
			t.Errorf("unexpected %s in payload: %v (expected %v)", field, payload[field], expected)
		}
	}

	// recipients_emails and columns are always sent so PATCH updates can clear them
	requiredFields := []string{"sonar_query_params", "recipients_emails", "columns"}
	for _, field := range requiredFields {
		if _, ok := payload[field]; !ok {
			t.Errorf("expected %s in payload", field)
		}
	}
}

func TestCreateScheduledReport(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertScheduledReportCreateRequest(t, req)

		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(testScheduledReportResponse)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	status := ScheduledReportStatusActive
	report, err := apiClient.CreateScheduledReport(ScheduledReport{
		Name:             "Weekly open alerts",
		Type:             "alerts_svl",
		Format:           "csv",
		Recurrence:       "weekly",
		FirstReportDate:  "2026-06-11T13:00:00Z",
		ExportTime:       "13:00:00",
		Status:           &status,
		SonarQuery:       `{"models":["Alert"],"type":"object_set"}`,
		SonarQueryParams: map[string]interface{}{"max_tier": 5},
		Columns:          []string{"OrcaScore", "Title"},
		RecipientsEmails: []string{"test@orca.security"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.ID != testScheduledReportID {
		t.Errorf("unexpected id: %s", report.ID)
	}
	if report.Status == nil || *report.Status != ScheduledReportStatusActive {
		t.Errorf("unexpected status: %v", report.Status)
	}
}

func TestGetScheduledReport(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "GET" {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/api/reporting/scheduled_reports/"+testScheduledReportID) {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(testScheduledReportResponse)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	report, err := apiClient.GetScheduledReport(testScheduledReportID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}
	if report.Name != "Weekly open alerts" {
		t.Errorf("unexpected name: %s", report.Name)
	}
	if report.Type != "alerts_svl" {
		t.Errorf("unexpected type: %s", report.Type)
	}
	if report.Recurrence != "weekly" {
		t.Errorf("unexpected recurrence: %s", report.Recurrence)
	}
	if report.SonarQuery != `{"models":["Alert"],"type":"object_set"}` {
		t.Errorf("unexpected sonar_query: %s", report.SonarQuery)
	}
	if report.Status == nil || *report.Status != ScheduledReportStatusActive {
		t.Errorf("unexpected status: %v", report.Status)
	}
	if len(report.RecipientsEmails) != 1 || report.RecipientsEmails[0] != "test@orca.security" {
		t.Errorf("unexpected recipients: %v", report.RecipientsEmails)
	}
}

func TestGetScheduledReport_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	report, err := apiClient.GetScheduledReport("invalid-id")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report != nil {
		t.Error("expected nil report for 404 response")
	}
}

func TestUpdateScheduledReport(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/api/reporting/scheduled_reports/"+testScheduledReportID) {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(testScheduledReportResponse)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	report, err := apiClient.UpdateScheduledReport(testScheduledReportID, ScheduledReport{
		Name:       "Weekly open alerts",
		Type:       "alerts_svl",
		Format:     "csv",
		Recurrence: "weekly",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.ID != testScheduledReportID {
		t.Errorf("unexpected id: %s", report.ID)
	}
}

// Deleting an optional attribute from a config leaves the plan value null, which
// reaches the client as the zero value. Updates are a PATCH, where an omitted key
// means "leave unchanged" — so every optional field must appear in the body at its
// zero value, or removing it from a config silently fails and each later refresh
// reports the stale server value as drift.
//
// The expected wire values are the ones the API accepts as "clear": "" for strings
// and {} for objects. Objects must not go out as null; the reporting API answers
// 500 to "config": null.
func TestScheduledReportPayload_SendsClearableZeroValuesForEveryOptionalField(t *testing.T) {
	wantEmptyString := []string{
		"sonar_query", "s3_path", "custom_email_subject", "custom_email_content",
		"bucket", "azure_blob_container", "google_cloud_storage_template", "snowflake_template",
	}
	wantEmptyObject := []string{
		"dsl_filter", "sonar_query_params", "query_filters", "config", "slack_channel",
	}

	// A report with nothing optional set: what Terraform sends once every optional
	// attribute has been removed from the config.
	cleared := ScheduledReport{
		Name:       "Weekly open alerts",
		Type:       "alerts_svl",
		Format:     "csv",
		Recurrence: "weekly",
		Columns:    []string{},
	}

	for _, verb := range []struct {
		name string
		send func(APIClient) error
	}{
		{"create", func(c APIClient) error { _, err := c.CreateScheduledReport(cleared); return err }},
		{"update", func(c APIClient) error {
			_, err := c.UpdateScheduledReport(testScheduledReportID, cleared)
			return err
		}},
	} {
		t.Run(verb.name, func(t *testing.T) {
			var raw []byte
			httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
				raw, _ = io.ReadAll(req.Body)
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(testScheduledReportResponse)),
					Request:    req,
				}
			})}
			if err := verb.send(newTestAPIClient(httpClient)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			payload := map[string]json.RawMessage{}
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Fatalf("request body is not valid JSON: %v", err)
			}

			for _, field := range wantEmptyString {
				got, ok := payload[field]
				if !ok {
					t.Errorf("%s is missing from the body; a removed attribute could never be cleared", field)
					continue
				}
				if string(got) != `""` {
					t.Errorf("%s = %s, want \"\"", field, got)
				}
			}
			for _, field := range wantEmptyObject {
				got, ok := payload[field]
				if !ok {
					t.Errorf("%s is missing from the body; a removed attribute could never be cleared", field)
					continue
				}
				if string(got) == "null" {
					t.Errorf("%s = null, which the API answers with 500; want {}", field)
					continue
				}
				if string(got) != "{}" {
					t.Errorf("%s = %s, want {}", field, got)
				}
			}
			// id and status are the two fields that keep omitempty: neither is ever
			// cleared, so sending their zero value would be wrong rather than useful.
			for _, field := range []string{"id", "status"} {
				if _, ok := payload[field]; ok {
					t.Errorf("%s should be omitted when unset, got %s", field, payload[field])
				}
			}
		})
	}
}

func TestDeleteScheduledReport(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/api/reporting/scheduled_reports/"+testScheduledReportID) {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}

		// the new reporting API returns 204 with an empty body
		return &http.Response{
			StatusCode: 204,
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	if err := apiClient.DeleteScheduledReport(testScheduledReportID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteScheduledReport_AlreadyDeleted(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	if err := apiClient.DeleteScheduledReport(testScheduledReportID); err != nil {
		t.Errorf("expected no error when report is already deleted, got: %v", err)
	}
}

func TestDoesScheduledReportExist_Found(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(testScheduledReportResponse)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	exists, err := apiClient.DoesScheduledReportExist(testScheduledReportID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected report to exist")
	}
}

func TestDoesScheduledReportExist_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	exists, err := apiClient.DoesScheduledReportExist("invalid-id")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected report to not exist")
	}
}
