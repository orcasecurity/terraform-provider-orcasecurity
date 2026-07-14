package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCreateDataDetectionRule_UsesPUTOnCollection(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		// rule create is PUT on the collection — not a mistake
		assertMethodPath(t, req, "PUT", "/api/scan_configuration/rules")
		assertCreateRulePayload(t, req)
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"rule_id":"rule-9"}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	ruleID, err := client.CreateDataDetectionRule(DataDetectionRule{
		Name:                  "tf rule",
		Feature:               "DSPM Scanning",
		Action:                "scan",
		Enabled:               false,
		SelectorCloudAccounts: []string{},
		SelectorBusinessUnits: []string{},
		Tags:                  []DataDetectionRuleTag{{Keys: []string{"*"}, Values: []string{"tf-managed"}}},
		Policies:              []string{"pol-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ruleID != "rule-9" {
		t.Errorf("expected rule-9, got %s", ruleID)
	}
}

func TestCreateDataDetectionRule_MissingRuleID(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	_, err := client.CreateDataDetectionRule(DataDetectionRule{Name: "tf rule", Feature: "DSPM Scanning", Action: "scan"})
	if err == nil {
		t.Fatal("expected error when response has no rule_id")
	}
}

func TestGetDataDetectionRule_Envelope(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "GET", "/api/scan_configuration/rules/rule-9")
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"rule_id":"rule-9","organization":"org-1","rule_name":"tf rule","feature":"DSPM Scanning","action":"scan","rule_priority":7,"is_enabled_rule":true,"is_default_rule":false,"tags":[{"keys":["*"],"values":["tf-managed"]}],"policies":["pol-1"],"selector_cloud_accounts":[],"selector_business_units":[]}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	rule, err := client.GetDataDetectionRule("rule-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule == nil || rule.ID != "rule-9" {
		t.Fatalf("expected rule-9, got %+v", rule)
	}
	if rule.Priority == nil || *rule.Priority != 7 {
		t.Errorf("expected priority 7, got %+v", rule.Priority)
	}
	if !rule.Enabled || rule.Action != "scan" || rule.OrganizationID != "org-1" {
		t.Errorf("unexpected rule: %+v", rule)
	}
	if len(rule.Policies) != 1 || rule.Policies[0] != "pol-1" {
		t.Errorf("unexpected policies: %+v", rule.Policies)
	}
}

func TestGetDataDetectionRule_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"status":"failure","errors":{"rule_id":["not found"]}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	rule, err := client.GetDataDetectionRule("missing")
	if err != nil {
		t.Fatalf("expected nil error on 404 so the resource can RemoveResource, got: %v", err)
	}
	if rule != nil {
		t.Errorf("expected nil rule on 404, got %+v", rule)
	}
}

func TestUpdateDataDetectionRule_UsesBulkRules(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		// rule update goes through the bulk endpoint — there is no PUT /rules/<id>
		assertMethodPath(t, req, "POST", "/api/scan_configuration/bulk_rules")
		assertBulkUpdatePayload(t, req)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	err := client.UpdateDataDetectionRule(DataDetectionRule{
		ID:                    "rule-9",
		Name:                  "tf rule renamed",
		Feature:               "DSPM Scanning",
		Action:                "scan",
		Enabled:               true,
		SelectorCloudAccounts: []string{},
		SelectorBusinessUnits: []string{},
		Tags:                  []DataDetectionRuleTag{{Keys: []string{"*"}, Values: []string{"tf-managed"}}},
		Policies:              []string{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteDataDetectionRule(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "DELETE", "/api/scan_configuration/rules/rule-9")
		return &http.Response{
			StatusCode: 204,
			Body:       io.NopCloser(strings.NewReader(``)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	if err := client.DeleteDataDetectionRule("rule-9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// assertCreateRulePayload validates the body PUT to the rules collection.
func assertCreateRulePayload(t *testing.T, req *http.Request) {
	t.Helper()
	body, _ := io.ReadAll(req.Body)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("invalid request body: %v", err)
	}
	if payload["rule_name"] != "tf rule" {
		t.Errorf("expected rule_name, got %v", payload["rule_name"])
	}
	if payload["feature"] != "DSPM Scanning" {
		t.Errorf("expected feature, got %v", payload["feature"])
	}
	if payload["action"] != "scan" {
		t.Errorf("expected action, got %v", payload["action"])
	}
	if payload["is_enabled_rule"] != false {
		t.Errorf("expected is_enabled_rule=false, got %v", payload["is_enabled_rule"])
	}
	if _, present := payload["rule_priority"]; present {
		t.Errorf("rule_priority must be omitted when nil so the server auto-assigns, got %v", payload["rule_priority"])
	}
	for _, key := range []string{"selector_cloud_accounts", "selector_business_units", "tags", "policies"} {
		if _, present := payload[key]; !present {
			t.Errorf("%s must always be serialized (bulk update is partial): %s", key, string(body))
		}
	}
	tags, _ := payload["tags"].([]interface{})
	if len(tags) != 1 {
		t.Fatalf("expected one tag selector, got %v", payload["tags"])
	}
	tag, _ := tags[0].(map[string]interface{})
	if keys, _ := tag["keys"].([]interface{}); len(keys) != 1 || keys[0] != "*" {
		t.Errorf("expected tag keys [*], got %v", tag["keys"])
	}
}

// assertBulkUpdatePayload validates the body POSTed to /bulk_rules.
func assertBulkUpdatePayload(t *testing.T, req *http.Request) {
	t.Helper()
	body, _ := io.ReadAll(req.Body)
	var payload struct {
		RulesToUpdate []map[string]interface{} `json:"rules_to_update"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("invalid request body: %v", err)
	}
	if len(payload.RulesToUpdate) != 1 {
		t.Fatalf("expected exactly one rule in rules_to_update, got %d", len(payload.RulesToUpdate))
	}
	rule := payload.RulesToUpdate[0]
	if rule["rule_id"] != "rule-9" {
		t.Errorf("expected rule_id in bulk payload, got %v", rule["rule_id"])
	}
	if rule["rule_name"] != "tf rule renamed" {
		t.Errorf("expected rule_name, got %v", rule["rule_name"])
	}
	// bulk update is partial: keys absent from the payload keep their
	// remote value, so every mutable list must always be present
	for _, key := range []string{"selector_cloud_accounts", "selector_business_units", "tags", "policies"} {
		if _, present := rule[key]; !present {
			t.Errorf("%s must always be serialized in bulk updates: %v", key, rule)
		}
	}
}
