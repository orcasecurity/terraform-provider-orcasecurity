package automation_v2

import (
	"io"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// roundTripFunc adapts a function into an http.RoundTripper. api_client.RoundTripFunc
// is defined in api_client's own _test.go file, so it is not visible outside that
// package's test binary; this is a local equivalent for use here.
type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func priorityTestResource(t *testing.T, responseJSON string, wantPath string) *automationV2Resource {
	t.Helper()
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) *http.Response {
		if wantPath != "" && req.URL.Path != wantPath {
			t.Errorf("expected path %s, got %s", wantPath, req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(responseJSON)),
			Request:    req,
		}
	})}
	return &automationV2Resource{apiClient: &api_client.APIClient{
		APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient,
	}}
}

func TestApplyPriorityReturnsServerValue(t *testing.T) {
	r := priorityTestResource(t,
		`{"status":"success","data":{"id":"a1","name":"n","status":"enabled","filter":{"sonar_query":{"models":["Alert"],"type":"object_set"}},"actions":[],"priority":3}}`,
		"/api/automations/a1/priority")
	actual, err := r.applyPriority("a1", 3)
	if err != nil {
		t.Fatalf("applyPriority failed: %v", err)
	}
	if actual != 3 {
		t.Errorf("expected 3, got %d", actual)
	}
}

func TestApplyPriorityReportsClampedValue(t *testing.T) {
	r := priorityTestResource(t,
		`{"status":"success","data":{"id":"a1","name":"n","status":"enabled","filter":{"sonar_query":{"models":["Alert"],"type":"object_set"}},"actions":[],"priority":10}}`,
		"")
	actual, err := r.applyPriority("a1", 50)
	if err != nil {
		t.Fatalf("applyPriority failed: %v", err)
	}
	if actual != 10 {
		t.Errorf("expected clamped 10, got %d", actual)
	}
}

func TestApplyPriorityErrorsWhenServerOmitsPriority(t *testing.T) {
	r := priorityTestResource(t,
		`{"status":"success","data":{"id":"a1","name":"n","status":"enabled","filter":{"sonar_query":{"models":["Alert"],"type":"object_set"}},"actions":[]}}`,
		"")
	_, err := r.applyPriority("a1", 2)
	if err == nil {
		t.Fatal("expected error when response has no priority, got nil")
	}
}

func TestClampErrorDetailMentionsBothValues(t *testing.T) {
	msg := clampErrorDetail(50, 10)
	if !strings.Contains(msg, "50") || !strings.Contains(msg, "10") {
		t.Errorf("clamp message must mention requested and actual values, got: %s", msg)
	}
}

func int64Ptr(v int64) *int64 { return &v }

func TestRefreshPriorityLeavesUntrackedNull(t *testing.T) {
	state := &automationV2ResourceModel{Priority: types.Int64Null()}
	refreshPriority(state, &api_client.AutomationV2{Priority: int64Ptr(4)})
	if !state.Priority.IsNull() {
		t.Errorf("untracked priority must stay null, got %v", state.Priority)
	}
}

func TestRefreshPriorityUpdatesTrackedValue(t *testing.T) {
	state := &automationV2ResourceModel{Priority: types.Int64Value(2)}
	refreshPriority(state, &api_client.AutomationV2{Priority: int64Ptr(4)})
	if state.Priority.ValueInt64() != 4 {
		t.Errorf("tracked priority must refresh to 4, got %v", state.Priority)
	}
}

func TestRefreshPriorityTrackedButServerOmitted(t *testing.T) {
	state := &automationV2ResourceModel{Priority: types.Int64Value(2)}
	refreshPriority(state, &api_client.AutomationV2{Priority: nil})
	if !state.Priority.IsNull() {
		t.Errorf("tracked priority with no server value must become null, got %v", state.Priority)
	}
}

func TestRefreshPriorityNilInstance(t *testing.T) {
	state := &automationV2ResourceModel{Priority: types.Int64Value(2)}
	refreshPriority(state, nil)
	if !state.Priority.IsNull() {
		t.Errorf("tracked priority with nil instance must become null, got %v", state.Priority)
	}
}
