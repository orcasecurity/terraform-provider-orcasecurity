package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestEnrichShiftLeftPolicyFromCatalog(t *testing.T) {
	catalogBody := `{
		"controls": [
			{
				"id": "ctrl-1",
				"title": "Catalog title",
				"priority": "MEDIUM",
				"disabled": false,
				"conditions": {"severities": {"operator": "IN", "values": ["CRITICAL"]}}
			}
		]
	}`

	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if strings.Contains(req.URL.Path, "/catalog/controls") {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(catalogBody)),
			}
		}
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader(`{}`))}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy := ShiftLeftPolicy{
		Controls: mustJSON([]map[string]interface{}{
			{"id": "ctrl-1", "priority": "HIGH", "disabled": true},
		}),
		PolicyData: mustJSON(map[string]interface{}{
			"controls": []map[string]interface{}{
				{"id": "ctrl-1", "priority": "HIGH", "disabled": true},
			},
		}),
	}

	if err := client.EnrichShiftLeftPolicyFromCatalog("iac", &policy); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var controls []map[string]interface{}
	if err := json.Unmarshal(policy.Controls, &controls); err != nil {
		t.Fatalf("failed to decode controls: %v", err)
	}
	if controls[0]["title"] != "Catalog title" {
		t.Errorf("expected catalog title, got %v", controls[0]["title"])
	}
	if controls[0]["priority"] != "HIGH" {
		t.Errorf("expected override priority HIGH, got %v", controls[0]["priority"])
	}
	if controls[0]["disabled"] != true {
		t.Errorf("expected override disabled true, got %v", controls[0]["disabled"])
	}
	if controls[0]["conditions"] == nil {
		t.Error("expected conditions from catalog")
	}
}

func TestEnrichShiftLeftPolicyFromCatalog_UnknownControl(t *testing.T) {
	catalogBody := `{"controls": [{"id": "ctrl-1", "title": "t"}]}`
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(catalogBody)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy := ShiftLeftPolicy{
		Controls: mustJSON([]map[string]interface{}{
			{"id": "missing", "priority": "HIGH", "disabled": true},
		}),
	}

	err := client.EnrichShiftLeftPolicyFromCatalog("iac", &policy)
	if err == nil || !strings.Contains(err.Error(), "unknown control id") {
		t.Fatalf("expected unknown control error, got %v", err)
	}
}

func mustJSON(v interface{}) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}
