package api_client_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"
)

func TestAutomationsV2_DoesAutomationV2Exist(t *testing.T) {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := apiClient.DoesAutomationV2Exist("1")
	if err != nil {
		t.Error(err)
	}
	if !exists {
		t.Error("automation v2 expected to exist, but it does not")
	}
}

func TestAutomationsV2_DoesAutomationV2ExistFalse(t *testing.T) {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := apiClient.DoesAutomationV2Exist("1")
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Error("automation v2 expected to be absent, but it exists")
	}
}

func TestAutomationsV2_GetAutomationV2(t *testing.T) {
	mockResponse := `{
		"status": "success",
		"data": {
			"id": "test-id-123",
			"name": "Test Automation",
			"description": "Test Description",
			"status": "enabled",
			"filter": {
				"sonar_query": {
					"models": ["Alert"],
					"type": "object_set"
				}
			},
			"actions": [
				{
					"id": "action-1",
					"type": 1,
					"data": {
						"external_config": "test-config-id"
					},
					"external_config": "test-config-id"
				}
			],
			"organization": "org-123",
			"end_time": "2024-12-31T23:59:59Z",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-02T12:00:00Z"
		}
	}`

	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		// Verify the request URL
		expectedURL := "/api/automations/test-id-123"
		if req.URL.Path != expectedURL {
			t.Errorf("Expected URL path %s, got %s", expectedURL, req.URL.Path)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(mockResponse)),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	automation, err := apiClient.GetAutomationV2("test-id-123")
	if err != nil {
		t.Fatalf("GetAutomationV2 failed: %v", err)
	}

	if automation == nil {
		t.Fatal("Expected automation to be returned, got nil")
	}

	if automation.ID != "test-id-123" {
		t.Errorf("Expected ID 'test-id-123', got '%s'", automation.ID)
	}
	if automation.Name != "Test Automation" {
		t.Errorf("Expected name 'Test Automation', got '%s'", automation.Name)
	}
	if automation.Status != "enabled" {
		t.Errorf("Expected status 'enabled', got '%s'", automation.Status)
	}
	if automation.EndTime != "2024-12-31T23:59:59Z" {
		t.Errorf("Expected end_time '2024-12-31T23:59:59Z', got '%s'", automation.EndTime)
	}
}

func TestAutomationsV2_GetAutomationV2NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error": "not found"}`)),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	automation, err := apiClient.GetAutomationV2("non-existent")
	if err == nil {
		t.Fatal("GetAutomationV2 should return error for 404")
	}

	if automation != nil {
		t.Error("Expected nil automation for 404 response, got non-nil")
	}

	// Verify it's a 404 error by checking the error message contains expected content
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Expected error to contain '404', got: %v", err)
	}
}

func TestAutomationsV2_CreateAutomationV2(t *testing.T) {
	expectedRequest := api_client.AutomationV2{
		Name:        "New Automation",
		Description: "New Description",
		Status:      "enabled",
		Filter: api_client.AutomationV2Filter{
			SonarQuery: api_client.AutomationV2SonarQuery{
				Models: []string{"Alert"},
				Type:   "object_set",
			},
		},
		Actions: []api_client.AutomationV2Action{
			{
				Type: 1,
				Data: map[string]interface{}{
					"external_config": "test-config-id",
				},
				ExternalConfig: stringPtr("test-config-id"),
			},
		},
		EndTime: "2024-12-31T23:59:59Z",
	}

	mockResponse := `{
		"status": "success",
		"data": {
			"id": "created-id-456",
			"name": "New Automation",
			"description": "New Description",
			"status": "enabled",
			"filter": {
				"sonar_query": {
					"models": ["Alert"],
					"type": "object_set"
				}
			},
			"actions": [
				{
					"id": "action-1",
					"type": 1,
					"data": {
						"external_config": "test-config-id"
					},
					"external_config": "test-config-id"
				}
			],
			"organization": "org-123",
			"end_time": "2024-12-31T23:59:59Z",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z"
		}
	}`

	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		// Verify the request method and URL
		if req.Method != "POST" {
			t.Errorf("Expected POST method, got %s", req.Method)
		}
		if req.URL.Path != "/api/automations" {
			t.Errorf("Expected URL path /api/automations, got %s", req.URL.Path)
		}

		return &http.Response{
			StatusCode: 201,
			Body:       ioutil.NopCloser(strings.NewReader(mockResponse)),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	automation, err := apiClient.CreateAutomationV2(expectedRequest)
	if err != nil {
		t.Fatalf("CreateAutomationV2 failed: %v", err)
	}

	if automation == nil {
		t.Fatal("Expected automation to be returned, got nil")
	}

	if automation.ID != "created-id-456" {
		t.Errorf("Expected ID 'created-id-456', got '%s'", automation.ID)
	}
	if automation.Name != "New Automation" {
		t.Errorf("Expected name 'New Automation', got '%s'", automation.Name)
	}
}

func TestAutomationsV2_UpdateAutomationV2(t *testing.T) {
	updateRequest := api_client.AutomationV2{
		ID:          "test-id-123",
		Name:        "Updated Automation",
		Description: "Updated Description",
		Status:      "disabled",
		Filter: api_client.AutomationV2Filter{
			SonarQuery: api_client.AutomationV2SonarQuery{
				Models: []string{"Alert"},
				Type:   "object_set",
			},
		},
		Actions: []api_client.AutomationV2Action{
			{
				Type: 1,
				Data: map[string]interface{}{
					"external_config": "updated-config-id",
				},
				ExternalConfig: stringPtr("updated-config-id"),
			},
		},
	}

	mockResponse := `{
		"status": "success",
		"data": {
			"id": "test-id-123",
			"name": "Updated Automation",
			"description": "Updated Description",
			"status": "disabled",
			"filter": {
				"sonar_query": {
					"models": ["Alert"],
					"type": "object_set"
				}
			},
			"actions": [
				{
					"id": "action-1",
					"type": 1,
					"data": {
						"external_config": "updated-config-id"
					},
					"external_config": "updated-config-id"
				}
			],
			"organization": "org-123",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-02T12:00:00Z"
		}
	}`

	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		// Verify the request method and URL
		if req.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", req.Method)
		}
		expectedURL := "/api/automations/test-id-123"
		if req.URL.Path != expectedURL {
			t.Errorf("Expected URL path %s, got %s", expectedURL, req.URL.Path)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(mockResponse)),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	automation, err := apiClient.UpdateAutomationV2("test-id-123", updateRequest)
	if err != nil {
		t.Fatalf("UpdateAutomationV2 failed: %v", err)
	}

	if automation == nil {
		t.Fatal("Expected automation to be returned, got nil")
	}

	if automation.Name != "Updated Automation" {
		t.Errorf("Expected name 'Updated Automation', got '%s'", automation.Name)
	}
	if automation.Status != "disabled" {
		t.Errorf("Expected status 'disabled', got '%s'", automation.Status)
	}
}

func TestAutomationsV2_DeleteAutomationV2(t *testing.T) {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		// Verify the request method and URL
		if req.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", req.Method)
		}
		expectedURL := "/api/automations/test-id-123"
		if req.URL.Path != expectedURL {
			t.Errorf("Expected URL path %s, got %s", expectedURL, req.URL.Path)
		}

		return &http.Response{
			StatusCode: 204,
			Body:       ioutil.NopCloser(strings.NewReader("")),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	err := apiClient.DeleteAutomationV2("test-id-123")
	if err != nil {
		t.Fatalf("DeleteAutomationV2 failed: %v", err)
	}
}

func TestAutomationV2_JSONMarshaling(t *testing.T) {
	// Test that our data structures can be properly marshaled and unmarshaled
	automation := api_client.AutomationV2{
		ID:          "test-id",
		Name:        "Test Automation",
		Description: "Test Description",
		Status:      "enabled",
		Filter: api_client.AutomationV2Filter{
			SonarQuery: api_client.AutomationV2SonarQuery{
				Models: []string{"Alert"},
				Type:   "object_set",
				With: map[string]interface{}{
					"operator": "and",
					"type":     "operation",
					"values": []interface{}{
						map[string]interface{}{
							"key":      "AlertType",
							"operator": "in",
							"type":     "str",
							"values":   []string{"test-alert-type"},
						},
					},
				},
			},
		},
		Actions: []api_client.AutomationV2Action{
			{
				ID:   "action-1",
				Type: 1,
				Data: map[string]interface{}{
					"external_config": "test-config",
					"type":            "LOGS",
				},
				ExternalConfig: stringPtr("test-config"),
			},
		},
		OrganizationID: "org-123",
		EndTime:        "2024-12-31T23:59:59Z",
		CreatedAt:      "2024-01-01T00:00:00Z",
		UpdatedAt:      "2024-01-02T12:00:00Z",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(automation)
	if err != nil {
		t.Fatalf("Failed to marshal automation to JSON: %v", err)
	}

	// Unmarshal back to struct
	var unmarshaledAutomation api_client.AutomationV2
	err = json.Unmarshal(jsonData, &unmarshaledAutomation)
	if err != nil {
		t.Fatalf("Failed to unmarshal automation from JSON: %v", err)
	}

	// Verify key fields
	if unmarshaledAutomation.ID != automation.ID {
		t.Errorf("ID mismatch after JSON round-trip: expected %s, got %s", automation.ID, unmarshaledAutomation.ID)
	}
	if unmarshaledAutomation.Name != automation.Name {
		t.Errorf("Name mismatch after JSON round-trip: expected %s, got %s", automation.Name, unmarshaledAutomation.Name)
	}
	if len(unmarshaledAutomation.Filter.SonarQuery.Models) != 1 || unmarshaledAutomation.Filter.SonarQuery.Models[0] != "Alert" {
		t.Errorf("SonarQuery models mismatch after JSON round-trip")
	}
	if unmarshaledAutomation.EndTime != automation.EndTime {
		t.Errorf("EndTime mismatch after JSON round-trip: expected %s, got %s", automation.EndTime, unmarshaledAutomation.EndTime)
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

