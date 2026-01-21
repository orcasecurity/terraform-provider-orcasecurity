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

// TestFlexibleString_Unmarshal tests that FlexibleString can unmarshal from both JSON numbers and strings
func TestFlexibleString_Unmarshal(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
	}{
		// Number types
		{name: "integer number", json: `{"value": 6}`, expected: "6"},
		{name: "float number", json: `{"value": 6.5}`, expected: "6.5"},
		{name: "zero", json: `{"value": 0}`, expected: "0"},
		{name: "negative integer", json: `{"value": -10}`, expected: "-10"},
		{name: "negative float", json: `{"value": -3.14}`, expected: "-3.14"},
		// String types
		{name: "numeric string", json: `{"value": "6"}`, expected: "6"},
		{name: "text string", json: `{"value": "hello"}`, expected: "hello"},
		{name: "empty string", json: `{"value": ""}`, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Value api_client.FlexibleString `json:"value"`
			}
			if err := json.Unmarshal([]byte(tt.json), &result); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if got := result.Value.String(); got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
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
		{name: "numeric value", value: api_client.FlexibleString("6"), expected: `{"value":"6"}`},
		{name: "text value", value: api_client.FlexibleString("hello"), expected: `{"value":"hello"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := struct {
				Value api_client.FlexibleString `json:"value"`
			}{Value: tt.value}

			result, err := json.Marshal(data)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if got := string(result); got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, got)
			}
		})
	}
}

// TestAutomationRange_Unmarshal tests automation range unmarshaling with both numbers and strings
func TestAutomationRange_Unmarshal(t *testing.T) {
	type rangeAssertion struct {
		field    string // gt, gte, lt, lte, eq
		expected string
	}

	tests := []struct {
		name       string
		jsonData   string
		assertions []rangeAssertion
	}{
		{
			name: "with numbers",
			jsonData: `{
				"data": {
					"dsl_filter": {
						"filter": [{
							"field": "state.orca_score",
							"range": {"gt": 6, "lte": 10}
						}]
					}
				}
			}`,
			assertions: []rangeAssertion{
				{field: "gt", expected: "6"},
				{field: "lte", expected: "10"},
			},
		},
		{
			name: "with strings",
			jsonData: `{
				"data": {
					"dsl_filter": {
						"filter": [{
							"field": "state.orca_score",
							"range": {"gte": "5", "lt": "8"}
						}]
					}
				}
			}`,
			assertions: []rangeAssertion{
				{field: "gte", expected: "5"},
				{field: "lt", expected: "8"},
			},
		},
		{
			name: "with mixed types",
			jsonData: `{
				"data": {
					"dsl_filter": {
						"filter": [{
							"field": "state.orca_score",
							"range": {"gte": 1, "lt": "100"}
						}]
					}
				}
			}`,
			assertions: []rangeAssertion{
				{field: "gte", expected: "1"},
				{field: "lt", expected: "100"},
			},
		},
	}

	type response struct {
		Data struct {
			DslFilter api_client.AutomationQuery `json:"dsl_filter"`
		} `json:"data"`
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp response
			if err := json.Unmarshal([]byte(tt.jsonData), &resp); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if len(resp.Data.DslFilter.Filter) != 1 {
				t.Fatalf("Expected 1 filter, got %d", len(resp.Data.DslFilter.Filter))
			}

			filter := resp.Data.DslFilter.Filter[0]
			if filter.Range == nil {
				t.Fatal("Expected range to be set")
			}

			// Assert each specified range field
			for _, assertion := range tt.assertions {
				var actual *api_client.FlexibleString
				switch assertion.field {
				case "gt":
					actual = filter.Range.Gt
				case "gte":
					actual = filter.Range.Gte
				case "lt":
					actual = filter.Range.Lt
				case "lte":
					actual = filter.Range.Lte
				case "eq":
					actual = filter.Range.Eq
				default:
					t.Fatalf("Unknown field: %s", assertion.field)
				}

				if actual == nil {
					t.Errorf("Expected %s to be set, but it was nil", assertion.field)
				} else if actual.String() != assertion.expected {
					t.Errorf("Expected %s to be %q, got %q", assertion.field, assertion.expected, actual.String())
				}
			}
		})
	}
}
