package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetDSPMPolicy(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "GET" {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/api/scan_configuration/dspm_policies/pol-1" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"policy_id":"pol-1","organization":"org-1","policy_name":"PII policy","policy_description":"desc","feature":"DSPM Scanning","tags":["team:sec"],"is_default_policy":false,"advanced_settings":{},"policy_document":{"selector_detectors":["AUS_TAX_NUMBER","det-1"],"selector_categories":["PII"],"selector_regions":[],"selector_industries":[],"selector_tags":[],"selector_countries":[]}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.GetDSPMPolicy("pol-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy == nil || policy.ID != "pol-1" {
		t.Fatalf("expected policy pol-1, got %+v", policy)
	}
	if policy.Name != "PII policy" || policy.Feature != "DSPM Scanning" || policy.OrganizationID != "org-1" {
		t.Errorf("unexpected fields: %+v", policy)
	}
	if len(policy.Document.SelectorDetectors) != 2 || policy.Document.SelectorDetectors[1] != "det-1" {
		t.Errorf("unexpected detectors: %+v", policy.Document.SelectorDetectors)
	}
	if len(policy.Tags) != 1 || policy.Tags[0] != "team:sec" {
		t.Errorf("unexpected tags: %+v", policy.Tags)
	}
}

func TestGetDSPMPolicy_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"status":"failure","errors":{"policy_id":["not found"]}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.GetDSPMPolicy("missing")
	if err != nil {
		t.Fatalf("expected nil error on 404 so the resource can RemoveResource, got: %v", err)
	}
	if policy != nil {
		t.Errorf("expected nil policy on 404, got %+v", policy)
	}
}

func TestCreateDSPMPolicy(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "POST" {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/scan_configuration/dspm_policies" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		body, _ := io.ReadAll(req.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if payload["policy_name"] != "PII policy" {
			t.Errorf("expected policy_name, got %v", payload["policy_name"])
		}
		// tags must serialize as a JSON array (never null) — server model expects a list
		if _, ok := payload["tags"].([]interface{}); !ok {
			t.Errorf("expected tags to be a JSON array, got %T (%v)", payload["tags"], payload["tags"])
		}
		if _, ok := payload["advanced_settings"].(map[string]interface{}); !ok {
			t.Errorf("expected advanced_settings to be a JSON object, got %T", payload["advanced_settings"])
		}
		doc, ok := payload["policy_document"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected policy_document object, got %T", payload["policy_document"])
		}
		if _, ok := doc["selector_detectors"].([]interface{}); !ok {
			t.Errorf("expected selector_detectors array, got %v", doc["selector_detectors"])
		}
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"policy_id":"pol-1","organization":"org-1","policy_name":"PII policy","policy_description":"desc","feature":"DSPM Scanning","tags":[],"is_default_policy":false,"policy_document":{"selector_detectors":["*"]}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.CreateDSPMPolicy(DSPMPolicy{
		Name:             "PII policy",
		Description:      "desc",
		Feature:          "DSPM Scanning",
		Tags:             []string{},
		AdvancedSettings: map[string]interface{}{},
		Document:         DSPMPolicyDocument{SelectorDetectors: []string{"*"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.ID != "pol-1" || policy.OrganizationID != "org-1" {
		t.Errorf("unexpected policy: %+v", policy)
	}
}

func TestUpdateDSPMPolicy(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "PUT" {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if req.URL.Path != "/api/scan_configuration/dspm_policies/pol-1" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"policy_id":"pol-1","organization":"org-1","policy_name":"Renamed","policy_description":"desc","feature":"DSPM Scanning","tags":[],"is_default_policy":false,"policy_document":{"selector_detectors":["*"]}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.UpdateDSPMPolicy("pol-1", DSPMPolicy{
		Name:             "Renamed",
		Description:      "desc",
		Feature:          "DSPM Scanning",
		Tags:             []string{},
		AdvancedSettings: map[string]interface{}{},
		Document:         DSPMPolicyDocument{SelectorDetectors: []string{"*"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Name != "Renamed" {
		t.Errorf("expected Renamed, got %s", policy.Name)
	}
}

func TestDeleteDSPMPolicy(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if req.URL.Path != "/api/scan_configuration/dspm_policies/pol-1" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 204,
			Body:       io.NopCloser(strings.NewReader(``)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	if err := client.DeleteDSPMPolicy("pol-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDSPMPolicy_EmptyDictTags(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		// policies created without tags (e.g. via the UI) carry tags: {} —
		// the server-side default is an empty dict, not a list
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"policy_id":"pol-1","policy_name":"ui policy","policy_description":"","tags":{},"policy_document":{"selector_detectors":["*"]}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.GetDSPMPolicy("pol-1")
	if err != nil {
		t.Fatalf("unexpected error decoding tags:{}: %v", err)
	}
	if policy == nil || policy.ID != "pol-1" {
		t.Fatalf("unexpected policy: %+v", policy)
	}
	if len(policy.Tags) != 0 {
		t.Errorf("expected empty tags, got %+v", policy.Tags)
	}
}

func TestPolicyTags_MarshalAsArray(t *testing.T) {
	payload, err := json.Marshal(DSPMPolicy{Name: "p", Tags: PolicyTags{}, AdvancedSettings: map[string]interface{}{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(payload), `"tags":[]`) {
		t.Errorf("tags must serialize as [], got: %s", string(payload))
	}
}
