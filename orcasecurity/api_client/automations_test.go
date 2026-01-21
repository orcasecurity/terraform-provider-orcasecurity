package api_client_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"
)

func TestAutomations_DoesAutomationExist(t *testing.T) {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := apiClient.DoesAutomationExist("1")
	if err != nil {
		t.Error(err)
	}
	if !exists {
		t.Error("automation expected to exists, but it does not")
	}

}
func TestAutomations_DoesAutomationExistFalse(t *testing.T) {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := apiClient.DoesAutomationExist("1")
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Error("automation expected to be absent, but it exists")
	}

}

// TestFlexibleString_UnmarshalNumber tests that FlexibleString can unmarshal from a JSON number
func TestFlexibleString_UnmarshalNumber(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
	}{
		{
			name:     "integer number",
			json:     `{"value": 6}`,
			expected: "6",
		},
		{
			name:     "float number",
			json:     `{"value": 6.5}`,
			expected: "6.5",
		},
		{
			name:     "zero",
			json:     `{"value": 0}`,
			expected: "0",
		},
		{
			name:     "negative integer",
			json:     `{"value": -10}`,
			expected: "-10",
		},
		{
			name:     "negative float",
			json:     `{"value": -3.14}`,
			expected: "-3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Value api_client.FlexibleString `json:"value"`
			}
			err := json.Unmarshal([]byte(tt.json), &result)
			if err != nil {
				t.Errorf("Unmarshal failed: %v", err)
				return
			}
			if result.Value.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.Value.String())
			}
		})
	}
}

// TestFlexibleString_UnmarshalString tests that FlexibleString can unmarshal from a JSON string
func TestFlexibleString_UnmarshalString(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
	}{
		{
			name:     "numeric string",
			json:     `{"value": "6"}`,
			expected: "6",
		},
		{
			name:     "text string",
			json:     `{"value": "hello"}`,
			expected: "hello",
		},
		{
			name:     "empty string",
			json:     `{"value": ""}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Value api_client.FlexibleString `json:"value"`
			}
			err := json.Unmarshal([]byte(tt.json), &result)
			if err != nil {
				t.Errorf("Unmarshal failed: %v", err)
				return
			}
			if result.Value.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.Value.String())
			}
		})
	}
}

// TestFlexibleString_Marshal tests that FlexibleString marshals as a string
func TestFlexibleString_Marshal(t *testing.T) {
	tests := []struct {
		name     string
		value    api_client.FlexibleString
		expected string
	}{
		{
			name:     "numeric value",
			value:    api_client.FlexibleString("6"),
			expected: `{"value":"6"}`,
		},
		{
			name:     "text value",
			value:    api_client.FlexibleString("hello"),
			expected: `{"value":"hello"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := struct {
				Value api_client.FlexibleString `json:"value"`
			}{
				Value: tt.value,
			}
			result, err := json.Marshal(data)
			if err != nil {
				t.Errorf("Marshal failed: %v", err)
				return
			}
			if string(result) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(result))
			}
		})
	}
}

// TestAutomationRange_UnmarshalWithNumbers tests the full automation range unmarshaling with numbers
func TestAutomationRange_UnmarshalWithNumbers(t *testing.T) {
	jsonData := `{
		"data": {
			"dsl_filter": {
				"filter": [
					{
						"field": "state.orca_score",
						"range": {
							"gt": 6,
							"lte": 10
						}
					}
				]
			}
		}
	}`

	type Response struct {
		Data struct {
			DslFilter api_client.AutomationQuery `json:"dsl_filter"`
		} `json:"data"`
	}

	var response Response
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(response.Data.DslFilter.Filter) != 1 {
		t.Errorf("Expected 1 filter, got %d", len(response.Data.DslFilter.Filter))
	}

	filter := response.Data.DslFilter.Filter[0]
	if filter.Range == nil {
		t.Fatal("Expected range to be set")
	}

	if filter.Range.Gt == nil || filter.Range.Gt.String() != "6" {
		t.Errorf("Expected gt to be '6', got %v", filter.Range.Gt)
	}

	if filter.Range.Lte == nil || filter.Range.Lte.String() != "10" {
		t.Errorf("Expected lte to be '10', got %v", filter.Range.Lte)
	}
}

// TestAutomationRange_UnmarshalWithStrings tests the full automation range unmarshaling with strings
func TestAutomationRange_UnmarshalWithStrings(t *testing.T) {
	jsonData := `{
		"data": {
			"dsl_filter": {
				"filter": [
					{
						"field": "state.orca_score",
						"range": {
							"gte": "5",
							"lt": "8"
						}
					}
				]
			}
		}
	}`

	type Response struct {
		Data struct {
			DslFilter api_client.AutomationQuery `json:"dsl_filter"`
		} `json:"data"`
	}

	var response Response
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(response.Data.DslFilter.Filter) != 1 {
		t.Errorf("Expected 1 filter, got %d", len(response.Data.DslFilter.Filter))
	}

	filter := response.Data.DslFilter.Filter[0]
	if filter.Range == nil {
		t.Fatal("Expected range to be set")
	}

	if filter.Range.Gte == nil || filter.Range.Gte.String() != "5" {
		t.Errorf("Expected gte to be '5', got %v", filter.Range.Gte)
	}

	if filter.Range.Lt == nil || filter.Range.Lt.String() != "8" {
		t.Errorf("Expected lt to be '8', got %v", filter.Range.Lt)
	}
}
