package automation_v2_priority_order

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// roundTripFunc adapts a function into an http.RoundTripper. api_client.RoundTripFunc
// is defined in api_client's own _test.go file, so it is not visible outside that
// package's test binary; this is a local equivalent for use here.
type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
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

func TestAssertOrderPutsSequentially(t *testing.T) {
	var puts []string // "id:priority" in call order
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) *http.Response {
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
	})}
	r := &automationPriorityOrderResource{apiClient: &api_client.APIClient{
		APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient,
	}}

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
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) *http.Response {
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
	})}
	r := &automationPriorityOrderResource{apiClient: &api_client.APIClient{
		APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient,
	}}

	err := r.assertOrder([]string{"ok", "gone"})
	if err == nil {
		t.Fatal("expected error for missing automation, got nil")
	}
	if !strings.Contains(err.Error(), "gone") {
		t.Errorf("error must name the failing automation ID, got: %v", err)
	}
}

func TestTopNIDs(t *testing.T) {
	body := `{"total_items": 3, "data": [` +
		automationJSON("x", 1) + `,` + automationJSON("y", 2) + `,` + automationJSON("z", 3) + `]}`
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Request: req}
	})}
	r := &automationPriorityOrderResource{apiClient: &api_client.APIClient{
		APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient,
	}}

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
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Request: req}
	})}
	r := &automationPriorityOrderResource{apiClient: &api_client.APIClient{
		APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient,
	}}

	ids, err := r.topNIDs(5)
	if err != nil {
		t.Fatalf("topNIDs failed: %v", err)
	}
	if len(ids) != 1 || ids[0] != "x" {
		t.Errorf("expected [x], got %v", ids)
	}
}
