package api_client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"
)

func newCustomTagRuleTestClient(handler func(req *http.Request) *http.Response) *api_client.APIClient {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(handler)}
	return &api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
}

func TestCustomTagRule_GetStringRule(t *testing.T) {
	apiClient := newCustomTagRuleTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"id": "rule-1",
					"name": "my rule",
					"description": "my description",
					"tags": {"environment": "production"},
					"rule": "AwsEc2Instance with PublicIps",
					"rule_type": "string",
					"disabled": false
				},
				"status": "success"
			}`)),
			Request: req,
		}
	})

	rule, err := apiClient.GetCustomTagRule("rule-1")
	if err != nil {
		t.Fatal(err)
	}
	if rule.ID != "rule-1" {
		t.Errorf("expected id 'rule-1', got %q", rule.ID)
	}
	if rule.Rule != "AwsEc2Instance with PublicIps" {
		t.Errorf("unexpected rule: %q", rule.Rule)
	}
	if rule.RuleType != "string" {
		t.Errorf("unexpected rule_type: %q", rule.RuleType)
	}
	if rule.Tags["environment"] != "production" {
		t.Errorf("unexpected tags: %v", rule.Tags)
	}
}

// The API returns the rule as a JSON object (not a string) when rule_type is
// "json". The client must normalize it back to a string.
func TestCustomTagRule_GetJsonRule(t *testing.T) {
	apiClient := newCustomTagRuleTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"id": "rule-2",
					"name": "my json rule",
					"tags": {"exposure": "public"},
					"rule": {"models": ["AwsEc2Instance"], "type": "object_set"},
					"rule_type": "json",
					"disabled": true
				},
				"status": "success"
			}`)),
			Request: req,
		}
	})

	rule, err := apiClient.GetCustomTagRule("rule-2")
	if err != nil {
		t.Fatal(err)
	}
	if !rule.Disabled {
		t.Error("expected rule to be disabled")
	}

	var decodedRule map[string]interface{}
	if err := json.Unmarshal([]byte(rule.Rule), &decodedRule); err != nil {
		t.Fatalf("rule is not valid JSON: %s, rule=%q", err, rule.Rule)
	}
	if decodedRule["type"] != "object_set" {
		t.Errorf("unexpected rule content: %q", rule.Rule)
	}
}

func TestCustomTagRule_Get404(t *testing.T) {
	apiClient := newCustomTagRuleTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"error": "not found", "status": "failure"}`)),
			Request:    req,
		}
	})

	rule, err := apiClient.GetCustomTagRule("missing")
	if err != nil {
		t.Fatal(err)
	}
	if rule != nil {
		t.Error("expected nil rule for 404 response")
	}
}

func TestCustomTagRule_DoesExist(t *testing.T) {
	for _, testCase := range []struct {
		statusCode int
		expected   bool
	}{
		{200, true},
		{404, false},
	} {
		apiClient := newCustomTagRuleTestClient(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: testCase.statusCode,
				Body:       io.NopCloser(strings.NewReader(`{"status": "success"}`)),
				Request:    req,
			}
		})

		exists, err := apiClient.DoesCustomTagRuleExist("rule-1")
		if err != nil {
			t.Fatal(err)
		}
		if exists != testCase.expected {
			t.Errorf("status %d: expected exists=%v, got %v", testCase.statusCode, testCase.expected, exists)
		}
	}
}

// The API expects the rule as a JSON object (not a string) when rule_type is
// "json". The client must decode the configured string before sending.
func TestCustomTagRule_CreateJsonRuleSendsRuleAsObject(t *testing.T) {
	var postedBody []byte
	apiClient := newCustomTagRuleTestClient(func(req *http.Request) *http.Response {
		if req.Method == "POST" {
			postedBody, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data": {"tags_rule_id": "new-rule-id"}, "status": "success"}`)),
				Request:    req,
			}
		}
		// follow-up GET after creation
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"id": "new-rule-id",
					"name": "my json rule",
					"tags": {"exposure": "public"},
					"rule": {"models": ["AwsEc2Instance"], "type": "object_set"},
					"rule_type": "json",
					"disabled": false
				},
				"status": "success"
			}`)),
			Request: req,
		}
	})

	rule, err := apiClient.CreateCustomTagRule(api_client.CustomTagRule{
		Name:     "my json rule",
		Tags:     map[string]string{"exposure": "public"},
		Rule:     `{"models": ["AwsEc2Instance"], "type": "object_set"}`,
		RuleType: "json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rule.ID != "new-rule-id" {
		t.Errorf("expected id 'new-rule-id', got %q", rule.ID)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(postedBody, &payload); err != nil {
		t.Fatalf("invalid POST payload: %s", err)
	}
	ruleField, isObject := payload["rule"].(map[string]interface{})
	if !isObject {
		t.Fatalf("expected rule to be sent as a JSON object, got %T: %v", payload["rule"], payload["rule"])
	}
	if ruleField["type"] != "object_set" {
		t.Errorf("unexpected rule payload: %v", ruleField)
	}
}

func TestCustomTagRule_CreateStringRuleSendsRuleAsString(t *testing.T) {
	var postedBody []byte
	apiClient := newCustomTagRuleTestClient(func(req *http.Request) *http.Response {
		if req.Method == "POST" {
			postedBody, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data": {"tags_rule_id": "new-rule-id"}, "status": "success"}`)),
				Request:    req,
			}
		}
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"id": "new-rule-id",
					"name": "my rule",
					"tags": {"exposure": "public"},
					"rule": "AwsEc2Instance with PublicIps",
					"rule_type": "string",
					"disabled": false
				},
				"status": "success"
			}`)),
			Request: req,
		}
	})

	_, err := apiClient.CreateCustomTagRule(api_client.CustomTagRule{
		Name:     "my rule",
		Tags:     map[string]string{"exposure": "public"},
		Rule:     "AwsEc2Instance with PublicIps",
		RuleType: "string",
	})
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(postedBody, &payload); err != nil {
		t.Fatalf("invalid POST payload: %s", err)
	}
	if _, isString := payload["rule"].(string); !isString {
		t.Errorf("expected rule to be sent as a string, got %T", payload["rule"])
	}
}

func TestCustomTagRule_CreateInvalidJsonRuleFailsBeforeRequest(t *testing.T) {
	requestMade := false
	apiClient := newCustomTagRuleTestClient(func(req *http.Request) *http.Response {
		requestMade = true
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status": "success"}`)),
			Request:    req,
		}
	})

	_, err := apiClient.CreateCustomTagRule(api_client.CustomTagRule{
		Name:     "broken",
		Tags:     map[string]string{"a": "b"},
		Rule:     "this is not json",
		RuleType: "json",
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON rule")
	}
	if requestMade {
		t.Error("no HTTP request should be made when the rule is invalid JSON")
	}
}

func TestCustomTagRule_UpdateParsesJsonRuleResponse(t *testing.T) {
	apiClient := newCustomTagRuleTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"id": "rule-1",
					"name": "updated",
					"tags": {"exposure": "public"},
					"rule": {"models": ["Vm"], "type": "object_set"},
					"rule_type": "json",
					"disabled": false
				},
				"status": "success"
			}`)),
			Request: req,
		}
	})

	rule, err := apiClient.UpdateCustomTagRule("rule-1", api_client.CustomTagRule{
		Name:     "updated",
		Tags:     map[string]string{"exposure": "public"},
		Rule:     `{"models": ["Vm"], "type": "object_set"}`,
		RuleType: "json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rule.Name != "updated" {
		t.Errorf("unexpected name: %q", rule.Name)
	}

	var decodedRule map[string]interface{}
	if err := json.Unmarshal([]byte(rule.Rule), &decodedRule); err != nil {
		t.Fatalf("rule is not valid JSON: %s, rule=%q", err, rule.Rule)
	}
}
