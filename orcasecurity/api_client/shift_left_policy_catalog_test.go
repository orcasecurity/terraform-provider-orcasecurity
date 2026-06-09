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

func TestAddAllCatalogControls_FlatScope(t *testing.T) {
	catalogBody := `{"controls":[
		{"id":"ctrl-1","title":"Control 1","priority":"HIGH"},
		{"id":"ctrl-2","title":"Control 2"}
	]}`
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(catalogBody)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy := ShiftLeftPolicy{}
	if err := client.AddAllCatalogControls("iac", &policy, []string{""}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var controls []map[string]interface{}
	if err := json.Unmarshal(policy.Controls, &controls); err != nil {
		t.Fatalf("failed to decode controls: %v", err)
	}
	if len(controls) != 2 {
		t.Fatalf("expected 2 controls injected, got %d", len(controls))
	}

	var policyData map[string]interface{}
	if err := json.Unmarshal(policy.PolicyData, &policyData); err != nil {
		t.Fatalf("failed to decode policy_data: %v", err)
	}
	if _, ok := policyData["controls"].([]interface{}); !ok {
		t.Error("expected controls at the top level of policy_data for flat scope")
	}

	// ctrl-2 has no priority in the catalog and should default to MEDIUM.
	for _, c := range controls {
		if c["id"] == "ctrl-2" && c["priority"] != "MEDIUM" {
			t.Errorf("expected default priority MEDIUM for ctrl-2, got %v", c["priority"])
		}
	}
}

func TestAddAllCatalogControls_ContainerScope(t *testing.T) {
	catalogBody := `{
		"vulnerabilities":{"controls":[{"id":"vuln-1","priority":"HIGH"}]},
		"secret_detection":{"controls":[{"id":"secret-1","priority":"LOW"}]}
	}`
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(catalogBody)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy := ShiftLeftPolicy{}
	if err := client.AddAllCatalogControls("container_image", &policy, []string{"vulnerabilities"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var policyData map[string]interface{}
	if err := json.Unmarshal(policy.PolicyData, &policyData); err != nil {
		t.Fatalf("failed to decode policy_data: %v", err)
	}
	scope, ok := policyData["vulnerabilities"].(map[string]interface{})
	if !ok {
		t.Fatal("expected vulnerabilities scope in policy_data")
	}
	if _, ok := scope["controls"].([]interface{}); !ok {
		t.Error("expected controls nested under vulnerabilities scope")
	}
	if _, ok := policyData["secret_detection"]; ok {
		t.Error("did not request secret_detection scope; it should not be injected")
	}

	var controls []map[string]interface{}
	if err := json.Unmarshal(policy.Controls, &controls); err != nil {
		t.Fatalf("failed to decode controls: %v", err)
	}
	if len(controls) != 1 || controls[0]["id"] != "vuln-1" {
		t.Errorf("expected only vuln-1 in flattened controls, got %+v", controls)
	}
}

func TestAddAllCatalogControls_NoScopes(t *testing.T) {
	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret"}
	policy := ShiftLeftPolicy{}
	if err := client.AddAllCatalogControls("iac", &policy, nil); err != nil {
		t.Fatalf("expected no error and no API call for empty scopes, got: %v", err)
	}
	if policy.Controls != nil || policy.PolicyData != nil {
		t.Error("expected policy to be untouched when no scopes are requested")
	}
}

func TestFlattenCatalogControls(t *testing.T) {
	catalogBody := json.RawMessage(`{"controls":[
		{"id":"ctrl-2","title":"Second","category":"cat-b","priority":"LOW"},
		{"id":"ctrl-1","title":"First","category":"cat-a","priority":"HIGH"}
	]}`)

	controls := FlattenCatalogControls(catalogBody)
	if len(controls) != 2 {
		t.Fatalf("expected 2 controls, got %d", len(controls))
	}
	// Results are sorted by ID.
	if controls[0].ID != "ctrl-1" || controls[1].ID != "ctrl-2" {
		t.Errorf("expected controls sorted by id, got %s, %s", controls[0].ID, controls[1].ID)
	}
	if controls[0].Title != "First" || controls[0].Category != "cat-a" || controls[0].Priority != "HIGH" {
		t.Errorf("unexpected summary for ctrl-1: %+v", controls[0])
	}
}

func mustJSON(v interface{}) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}
