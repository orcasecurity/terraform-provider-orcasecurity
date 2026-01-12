package api_client_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"
)

const GENERIC_ERR_TEMPLATE = "expected no error, got: %v"
const GTE_FIVE_ERR_TEMPLATE = "expected Gte to be '5', got: %v"

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

func TestAutomationRange_UnmarshalJSON_NumericValues(t *testing.T) {
	// Test case: API returns numeric values (the bug scenario)
	jsonData := `{"gte": 5, "lte": 10, "gt": 3, "lt": 15, "eq": 7}`

	var r api_client.AutomationRange
	err := json.Unmarshal([]byte(jsonData), &r)
	if err != nil {
		t.Fatalf(GENERIC_ERR_TEMPLATE, err)
	}

	if r.Gte == nil || *r.Gte != "5" {
		t.Errorf(GTE_FIVE_ERR_TEMPLATE, r.Gte)
	}
	if r.Lte == nil || *r.Lte != "10" {
		t.Errorf("expected Lte to be '10', got: %v", r.Lte)
	}
	if r.Gt == nil || *r.Gt != "3" {
		t.Errorf("expected Gt to be '3', got: %v", r.Gt)
	}
	if r.Lt == nil || *r.Lt != "15" {
		t.Errorf("expected Lt to be '15', got: %v", r.Lt)
	}
	if r.Eq == nil || *r.Eq != "7" {
		t.Errorf("expected Eq to be '7', got: %v", r.Eq)
	}
}

func TestAutomationRange_UnmarshalJSON_StringValues(t *testing.T) {
	// Test case: API returns string values (backward compatibility)
	jsonData := `{"gte": "5", "lte": "10"}`

	var r api_client.AutomationRange
	err := json.Unmarshal([]byte(jsonData), &r)
	if err != nil {
		t.Fatalf(GENERIC_ERR_TEMPLATE, err)
	}

	if r.Gte == nil || *r.Gte != "5" {
		t.Errorf(GTE_FIVE_ERR_TEMPLATE, r.Gte)
	}
	if r.Lte == nil || *r.Lte != "10" {
		t.Errorf("expected Lte to be '10', got: %v", r.Lte)
	}
}

func TestAutomationRange_UnmarshalJSON_NullValues(t *testing.T) {
	// Test case: Some fields are null or missing
	jsonData := `{"gte": 5, "lte": null}`

	var r api_client.AutomationRange
	err := json.Unmarshal([]byte(jsonData), &r)
	if err != nil {
		t.Fatalf(GENERIC_ERR_TEMPLATE, err)
	}

	if r.Gte == nil || *r.Gte != "5" {
		t.Errorf(GTE_FIVE_ERR_TEMPLATE, r.Gte)
	}
	if r.Lte != nil {
		t.Errorf("expected Lte to be nil, got: %v", *r.Lte)
	}
	if r.Gt != nil {
		t.Errorf("expected Gt to be nil, got: %v", *r.Gt)
	}
}

func TestAutomationRange_UnmarshalJSON_FloatValues(t *testing.T) {
	// Test case: API returns float values
	jsonData := `{"gte": 3.5, "lte": 7.25}`

	var r api_client.AutomationRange
	err := json.Unmarshal([]byte(jsonData), &r)
	if err != nil {
		t.Fatalf(GENERIC_ERR_TEMPLATE, err)
	}

	if r.Gte == nil || *r.Gte != "3.5" {
		t.Errorf("expected Gte to be '3.5', got: %v", r.Gte)
	}
	if r.Lte == nil || *r.Lte != "7.25" {
		t.Errorf("expected Lte to be '7.25', got: %v", r.Lte)
	}
}

func TestAutomationRange_UnmarshalJSON_EmptyObject(t *testing.T) {
	// Test case: Empty object
	jsonData := `{}`

	var r api_client.AutomationRange
	err := json.Unmarshal([]byte(jsonData), &r)
	if err != nil {
		t.Fatalf(GENERIC_ERR_TEMPLATE, err)
	}

	if r.Gte != nil {
		t.Errorf("expected Gte to be nil, got: %v", *r.Gte)
	}
	if r.Lte != nil {
		t.Errorf("expected Lte to be nil, got: %v", *r.Lte)
	}
}