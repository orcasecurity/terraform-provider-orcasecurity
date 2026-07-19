package automation_v2

import (
	"io"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func priorityTestResource(t *testing.T, responseJSON string, wantPath string) *automationV2Resource {
	t.Helper()
	apiClient := testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		if wantPath != "" && req.URL.Path != wantPath {
			t.Errorf("expected path %s, got %s", wantPath, req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(responseJSON)),
			Request:    req,
		}
	})
	return &automationV2Resource{apiClient: apiClient}
}

func int64Ptr(v int64) *int64 { return &v }

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

// Create priority failures must be warnings, never errors: a Create error
// would taint the freshly created automation and force replacement.
func TestApplyPlanPriorityOnCreateFailureWarnsAndKeepsAutomation(t *testing.T) {
	r := &automationV2Resource{apiClient: testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader(`{"status":"failure","message":"boom"}`)),
			Request:    req,
		}
	})}
	plan := &automationV2ResourceModel{Priority: types.Int64Value(3)}
	resp := &resource.CreateResponse{}

	r.applyPlanPriorityOnCreate(plan, "a1", resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("priority failure on create must not be an error (taints the resource), got: %v", resp.Diagnostics)
	}
	if resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("expected exactly 1 warning, got %d", resp.Diagnostics.WarningsCount())
	}
	if !plan.Priority.IsNull() {
		t.Errorf("failed priority must be nulled in the plan so the next apply retries, got %v", plan.Priority)
	}
}

func TestApplyPlanPriorityOnCreateClampWarnsWithActual(t *testing.T) {
	r := priorityTestResource(t,
		`{"status":"success","data":{"id":"a1","name":"n","status":"enabled","filter":{"sonar_query":{"models":["Alert"],"type":"object_set"}},"actions":[],"priority":10}}`,
		"")
	plan := &automationV2ResourceModel{Priority: types.Int64Value(50)}
	resp := &resource.CreateResponse{}

	r.applyPlanPriorityOnCreate(plan, "a1", resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("clamp on create must not be an error, got: %v", resp.Diagnostics)
	}
	if resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("expected exactly 1 warning, got %d", resp.Diagnostics.WarningsCount())
	}
	if plan.Priority.ValueInt64() != 10 {
		t.Errorf("plan must record the server's actual priority 10, got %v", plan.Priority)
	}
}

func TestModelsEqualIgnoringPriority(t *testing.T) {
	base := automationV2ResourceModel{
		ID:       types.StringValue("a1"),
		Name:     types.StringValue("n"),
		Priority: types.Int64Value(1),
	}
	other := base
	other.Priority = types.Int64Value(7)
	if !modelsEqualIgnoringPriority(base, other) {
		t.Error("models differing only in priority must be equal")
	}
	other.Name = types.StringValue("renamed")
	if modelsEqualIgnoringPriority(base, other) {
		t.Error("models differing in name must not be equal")
	}
}
