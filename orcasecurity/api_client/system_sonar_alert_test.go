package api_client

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestGetSystemSonarAlert(t *testing.T) {
	mockResponse := `{
		"data": {
			"rule_id": "r8ae477067a",
			"rule_type": "apigateway_routes_without_authorization_type",
			"name": "API Gateway Route is not configured with an authorization type",
			"category": "Authentication",
			"score": 4,
			"enabled": true,
			"custom": false
		}
	}`

	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "GET" {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/api/sonar/rules/r8ae477067a") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(mockResponse)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	alert, err := apiClient.GetSystemSonarAlert("r8ae477067a")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alert.RuleID != "r8ae477067a" {
		t.Errorf("expected rule_id r8ae477067a, got %s", alert.RuleID)
	}
	if alert.Name != "API Gateway Route is not configured with an authorization type" {
		t.Errorf("unexpected name: %s", alert.Name)
	}
	if alert.Category != "Authentication" {
		t.Errorf("expected category Authentication, got %s", alert.Category)
	}
	if alert.Score != 4 {
		t.Errorf("expected score 4, got %f", alert.Score)
	}
}

func TestGetSystemSonarAlert_NotFound(t *testing.T) {
	mockResponse := `{"error": "not found"}`

	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       ioutil.NopCloser(strings.NewReader(mockResponse)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	_, err := apiClient.GetSystemSonarAlert("invalid-id")

	if err == nil {
		t.Error("expected error for not found alert")
	}
}

func TestUpdateSystemSonarAlertStatus(t *testing.T) {
	mockResponse := `{
		"version": "0.1.0",
		"rule_id": "r8ae477067a",
		"enabled": false,
		"status": "success"
	}`

	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "PUT" {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/api/sonar/rules/status/r8ae477067a") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(mockResponse)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, err := apiClient.UpdateSystemSonarAlertStatus("r8ae477067a", "apigateway_routes_without_authorization_type", false)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.RuleID != "r8ae477067a" {
		t.Errorf("expected rule_id r8ae477067a, got %s", resp.RuleID)
	}
	if resp.Enabled != false {
		t.Errorf("expected enabled false, got %v", resp.Enabled)
	}
	if resp.Status != "success" {
		t.Errorf("expected status success, got %s", resp.Status)
	}
}

func TestUpdateSystemSonarAlertStatus_Enable(t *testing.T) {
	mockResponse := `{
		"version": "0.1.0",
		"rule_id": "r8ae477067a",
		"enabled": true,
		"status": "success"
	}`

	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(mockResponse)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, err := apiClient.UpdateSystemSonarAlertStatus("r8ae477067a", "apigateway_routes_without_authorization_type", true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Enabled != true {
		t.Errorf("expected enabled true, got %v", resp.Enabled)
	}
}

func TestDoesSystemSonarAlertExist_Found(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "HEAD" {
			t.Errorf("expected HEAD, got %s", req.Method)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := apiClient.DoesSystemSonarAlertExist("r8ae477067a")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected alert to exist")
	}
}

func TestDoesSystemSonarAlertExist_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error": "not found"}`)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := apiClient.DoesSystemSonarAlertExist("invalid-id")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected alert to not exist")
	}
}

func TestDoesSystemSonarAlertExist_ServerError(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 500,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error": "internal server error"}`)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := apiClient.DoesSystemSonarAlertExist("r8ae477067a")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected alert to not exist on server error")
	}
}
