package automation_v2_priority_order

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// stubResource returns a resource whose API requests are answered by fn.
func stubResource(fn testutils.RoundTripFunc) *automationPriorityOrderResource {
	return &automationPriorityOrderResource{apiClient: testutils.NewStubAPIClient(fn)}
}

func automationJSON(id string, priority int) string {
	return fmt.Sprintf(
		`{"id":"%s","name":"auto-%s","status":"enabled","filter":{"sonar_query":{"models":["Alert"],"type":"object_set"}},"actions":[],"priority":%d}`,
		id, id, priority)
}

func TestSchemaAutomationIDsRequired(t *testing.T) {
	r := &automationPriorityOrderResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["automation_ids"].(schema.ListAttribute)
	if !ok {
		t.Fatal("automation_ids is not a ListAttribute")
	}
	if !attr.Required {
		t.Error("automation_ids must be Required")
	}
	if len(attr.Validators) != 2 {
		t.Errorf("automation_ids must have 2 validators (SizeAtLeast, UniqueValues), got %d", len(attr.Validators))
	}
}

// id is the fixed singleton marker; it must be Computed and carry
// UseStateForUnknown so re-applies do not plan "(known after apply)".
func TestSchemaIDComputedWithUseStateForUnknown(t *testing.T) {
	r := &automationPriorityOrderResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["id"].(schema.StringAttribute)
	if !ok {
		t.Fatal("id is not a StringAttribute")
	}
	if !attr.Computed {
		t.Error("id must be Computed")
	}
	if len(attr.PlanModifiers) != 1 {
		t.Errorf("id must have exactly 1 plan modifier (UseStateForUnknown), got %d", len(attr.PlanModifiers))
	}
}

func TestAssertOrderPutsSequentially(t *testing.T) {
	var puts []string // "id:priority" in call order
	r := stubResource(func(req *http.Request) *http.Response {
		if req.Method != http.MethodPut || !strings.HasSuffix(req.URL.Path, "/priority") {
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
		}
		body, _ := io.ReadAll(req.Body)
		var payload struct {
			Priority int64 `json:"priority"`
		}
		_ = json.Unmarshal(body, &payload)
		id := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/api/automations/"), "/priority")
		puts = append(puts, fmt.Sprintf("%s:%d", id, payload.Priority))
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(
				`{"status":"success","data":` + automationJSON(id, int(payload.Priority)) + `}`)),
			Request: req,
		}
	})

	err := r.assertOrder([]string{"b", "a", "c"})
	if err != nil {
		t.Fatalf("assertOrder failed: %v", err)
	}
	want := []string{"b:1", "a:2", "c:3"}
	if strings.Join(puts, ",") != strings.Join(want, ",") {
		t.Errorf("expected PUTs %v in order, got %v", want, puts)
	}
}

func TestAssertOrderSurfacesFailingID(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		if strings.Contains(req.URL.Path, "/gone/") {
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader(`{"status":"failure","message":"not found"}`)),
				Request:    req,
			}
		}
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(
				`{"status":"success","data":` + automationJSON("ok", 1) + `}`)),
			Request: req,
		}
	})

	err := r.assertOrder([]string{"ok", "gone"})
	if err == nil {
		t.Fatal("expected error for missing automation, got nil")
	}
	if !strings.Contains(err.Error(), "gone") {
		t.Errorf("error must name the failing automation ID, got: %v", err)
	}
}

// orderStub answers priority PUTs with success and list GETs with the given
// server order, so applyOrder's verification can be exercised.
func orderStub(serverOrder ...string) *automationPriorityOrderResource {
	return stubResource(func(req *http.Request) *http.Response {
		if req.Method == http.MethodPut {
			id := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/api/automations/"), "/priority")
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(strings.NewReader(
					`{"status":"success","data":` + automationJSON(id, 1) + `}`)),
				Request: req,
			}
		}
		items := make([]string, 0, len(serverOrder))
		for i, id := range serverOrder {
			items = append(items, automationJSON(id, i+1))
		}
		body := fmt.Sprintf(`{"total_items": %d, "data": [%s]}`, len(serverOrder), strings.Join(items, ","))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Request: req}
	})
}

func TestApplyOrderVerifiesAchievedOrder(t *testing.T) {
	r := orderStub("a", "b")
	if err := r.applyOrder([]string{"a", "b"}); err != nil {
		t.Fatalf("applyOrder must succeed when the server converges, got: %v", err)
	}
}

// Legacy duplicate priorities can make an order unreachable: every PUT
// "succeeds" but the achieved order differs. applyOrder must fail with the
// achieved order instead of silently saving the desired state.
func TestApplyOrderFailsWhenServerDoesNotConverge(t *testing.T) {
	r := orderStub("a", "x")
	err := r.applyOrder([]string{"a", "c"})
	if err == nil {
		t.Fatal("expected convergence error, got nil")
	}
	if !strings.Contains(err.Error(), "x") || !strings.Contains(err.Error(), "c") {
		t.Errorf("error must report requested and achieved orders, got: %v", err)
	}
}

func TestTopNIDs(t *testing.T) {
	body := `{"total_items": 3, "data": [` +
		automationJSON("x", 1) + `,` + automationJSON("y", 2) + `,` + automationJSON("z", 3) + `]}`
	r := stubResource(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Request: req}
	})

	ids, err := r.topNIDs(2)
	if err != nil {
		t.Fatalf("topNIDs failed: %v", err)
	}
	if len(ids) != 2 || ids[0] != "x" || ids[1] != "y" {
		t.Errorf("expected [x y], got %v", ids)
	}
}

func TestTopNIDsFewerThanN(t *testing.T) {
	body := `{"total_items": 1, "data": [` + automationJSON("x", 1) + `]}`
	r := stubResource(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Request: req}
	})

	ids, err := r.topNIDs(5)
	if err != nil {
		t.Fatalf("topNIDs failed: %v", err)
	}
	if len(ids) != 1 || ids[0] != "x" {
		t.Errorf("expected [x], got %v", ids)
	}
}
