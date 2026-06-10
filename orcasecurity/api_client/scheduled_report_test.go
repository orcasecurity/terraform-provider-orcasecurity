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

func TestCreateScheduledReport(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
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
		if payload["name"] != "Weekly open alerts" {
			t.Errorf("unexpected name in payload: %v", payload["name"])
		}
		if payload["type"] != "alerts_svl" {
			t.Errorf("expected string enum for type, got %v", payload["type"])
		}
		if payload["status"] != float64(1) {
			t.Errorf("expected integer status 1, got %v", payload["status"])
		}
		if _, ok := payload["sonar_query_params"]; !ok {
			t.Error("expected sonar_query_params in payload")
		}
		// always sent so PATCH updates can clear them
		if _, ok := payload["recipients_emails"]; !ok {
			t.Error("expected recipients_emails in payload")
		}
		if _, ok := payload["columns"]; !ok {
			t.Error("expected columns in payload")
		}

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
