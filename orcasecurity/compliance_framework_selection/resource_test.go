package compliance_framework_selection

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDiffScopes(t *testing.T) {
	tests := []struct {
		name       string
		from, to   []string
		wantAdd    []string
		wantRemove []string
	}{
		{"empty to user", nil, []string{"user"}, []string{"user"}, nil},
		{"user to user+org", []string{"user"}, []string{"user", "organization"}, []string{"organization"}, nil},
		{"user+org to org", []string{"user", "organization"}, []string{"organization"}, nil, []string{"user"}},
		{"org to empty", []string{"organization"}, nil, nil, []string{"organization"}},
		{"swap org to user", []string{"organization"}, []string{"user"}, []string{"user"}, []string{"organization"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			add, remove := DiffScopes(tt.from, tt.to)
			if !testutils.SameElements(add, tt.wantAdd) {
				t.Errorf("add = %v, want %v", add, tt.wantAdd)
			}
			if !testutils.SameElements(remove, tt.wantRemove) {
				t.Errorf("remove = %v, want %v", remove, tt.wantRemove)
			}
		})
	}
}

func stubResource(fn testutils.RoundTripFunc) *complianceFrameworkSelectionResource {
	return &complianceFrameworkSelectionResource{apiClient: testutils.NewStubAPIClient(fn)}
}

func resourceSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &resource.SchemaResponse{}
	(&complianceFrameworkSelectionResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema build failed: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func model(t *testing.T, frameworkID string, scopes []string) resourceModel {
	t.Helper()
	return resourceModel{
		ID:          types.StringValue(frameworkID),
		FrameworkID: types.StringValue(frameworkID),
		Scopes:      testutils.StringSet(t, scopes...),
		DisplayName: types.StringValue("Lab"),
		Custom:      types.BoolValue(false),
		IsReady:     types.BoolValue(true),
	}
}

func stateWith(t *testing.T, sch schema.Schema, m resourceModel) tfsdk.State {
	t.Helper()
	st := tfsdk.State{Schema: sch}
	if diags := st.Set(context.Background(), &m); diags.HasError() {
		t.Fatalf("failed to seed state: %v", diags)
	}
	return st
}

func planWith(t *testing.T, sch schema.Schema, m resourceModel) tfsdk.Plan {
	t.Helper()
	p := tfsdk.Plan{Schema: sch}
	if diags := p.Set(context.Background(), &m); diags.HasError() {
		t.Fatalf("failed to seed plan: %v", diags)
	}
	return p
}

type httpCall struct {
	Method string
	Path   string
	Scope  string
}

func selectStub(t *testing.T, entries map[string]map[string]interface{}, record *[]httpCall, statusFor func(*http.Request) int) testutils.RoundTripFunc {
	t.Helper()
	return func(req *http.Request) *http.Response {
		var payload struct {
			FrameworkIDs []string `json:"framework_ids"`
			Scope        string   `json:"scope"`
		}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(body))
			_ = json.Unmarshal(body, &payload)
		}
		if record != nil && req.URL.Path == "/api/compliance/frameworks/select" && req.Method != "GET" {
			*record = append(*record, httpCall{Method: req.Method, Path: req.URL.Path, Scope: payload.Scope})
		}
		code := 200
		if statusFor != nil {
			code = statusFor(req)
		}
		if req.Method == "GET" {
			raw, _ := json.Marshal(entries)
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Request: req}
		}
		if code != 200 {
			return &http.Response{
				StatusCode: code,
				Body:       io.NopCloser(strings.NewReader(`{"error":"Framework X can't be selected as it doesn't exist."}`)),
				Request:    req,
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}
	}
}

func TestRead_AbsentRemovesResource(t *testing.T) {
	r := stubResource(selectStub(t, map[string]map[string]interface{}{}, nil, nil))
	sch := resourceSchema(t)
	m := model(t, "gone", []string{"user"})
	req := resource.ReadRequest{State: stateWith(t, sch, m)}
	resp := &resource.ReadResponse{State: stateWith(t, sch, m)}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("absent framework must not error, got: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("expected RemoveResource, got %v", resp.State.Raw)
	}
}

func TestRead_EmptyScopesKeepsResource(t *testing.T) {
	entries := map[string]map[string]interface{}{
		"cost_optimization": {
			"id": "cost_optimization", "active": false, "selection_scopes": []string{},
			"display_name": "Cost Optimization", "custom": false, "is_ready": true,
		},
	}
	r := stubResource(selectStub(t, entries, nil, nil))
	sch := resourceSchema(t)
	m := model(t, "cost_optimization", []string{"user"})
	req := resource.ReadRequest{State: stateWith(t, sch, m)}
	resp := &resource.ReadResponse{State: stateWith(t, sch, m)}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("empty scopes must not error: %v", resp.Diagnostics)
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("empty scopes must keep the resource in state")
	}
	var out resourceModel
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatalf("decode: %v", diags)
	}
	if out.Scopes.IsNull() {
		t.Fatal("scopes must be an empty set, not null")
	}
	if len(out.Scopes.Elements()) != 0 {
		t.Errorf("scopes = %v, want empty", out.Scopes.Elements())
	}
}

func TestUpdate_ScopeDiffs(t *testing.T) {
	tests := []struct {
		name      string
		from, to  []string
		wantCalls []httpCall
	}{
		{"empty to user", nil, []string{"user"}, []httpCall{{"POST", "/api/compliance/frameworks/select", "user"}}},
		{"user to user+org", []string{"user"}, []string{"user", "organization"}, []httpCall{{"POST", "/api/compliance/frameworks/select", "organization"}}},
		{"user+org to org", []string{"user", "organization"}, []string{"organization"}, []httpCall{{"DELETE", "/api/compliance/frameworks/select", "user"}}},
		{"org to empty", []string{"organization"}, nil, []httpCall{{"DELETE", "/api/compliance/frameworks/select", "organization"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []httpCall
			entries := map[string]map[string]interface{}{
				"fw": {
					"id": "fw", "active": len(tt.to) > 0, "selection_scopes": tt.to,
					"display_name": "FW", "custom": false, "is_ready": true,
				},
			}
			r := stubResource(selectStub(t, entries, &calls, nil))
			sch := resourceSchema(t)
			from := model(t, "fw", tt.from)
			to := model(t, "fw", tt.to)
			req := resource.UpdateRequest{State: stateWith(t, sch, from), Plan: planWith(t, sch, to)}
			resp := &resource.UpdateResponse{State: stateWith(t, sch, from)}
			r.Update(context.Background(), req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("update: %v", resp.Diagnostics)
			}
			if len(calls) != len(tt.wantCalls) {
				t.Fatalf("calls = %#v, want %#v", calls, tt.wantCalls)
			}
			for i := range calls {
				if calls[i] != tt.wantCalls[i] {
					t.Errorf("call %d = %#v, want %#v", i, calls[i], tt.wantCalls[i])
				}
			}
		})
	}
}

func TestDelete_DefaultIssuesNoHTTP(t *testing.T) {
	var calls []httpCall
	r := stubResource(selectStub(t, map[string]map[string]interface{}{}, &calls, nil))
	sch := resourceSchema(t)
	m := model(t, "fw", []string{"user"})
	req := resource.DeleteRequest{State: stateWith(t, sch, m)}
	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete: %v", resp.Diagnostics)
	}
	if len(calls) != 0 {
		t.Errorf("default destroy must issue no HTTP, got %#v", calls)
	}
}

func TestCreate_PersonalRejectsOrganization(t *testing.T) {
	var calls []httpCall
	entries := map[string]map[string]interface{}{
		"fw": {
			"id": "fw", "active": false, "selection_scopes": []string{},
			"display_name": "FW", "custom": true, "is_ready": true,
			"visibility": "Personal",
		},
	}
	r := stubResource(selectStub(t, entries, &calls, nil))
	sch := resourceSchema(t)
	m := model(t, "fw", []string{"organization"})
	req := resource.CreateRequest{Plan: planWith(t, sch, m)}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("personal + organization must be a diagnostic")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Summary(), api_client.ErrPersonalOrgSummary) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected personal/org diagnostic, got %v", resp.Diagnostics)
	}
	if len(calls) != 0 {
		t.Errorf("must not call select, got %#v", calls)
	}
}

func TestUpdate_PersonalRejectsOrganization(t *testing.T) {
	var calls []httpCall
	entries := map[string]map[string]interface{}{
		"fw": {
			"id": "fw", "active": true, "selection_scopes": []string{"user"},
			"display_name": "FW", "custom": true, "is_ready": true,
			"visibility": "Personal",
		},
	}
	r := stubResource(selectStub(t, entries, &calls, nil))
	sch := resourceSchema(t)
	from := model(t, "fw", []string{"user"})
	to := model(t, "fw", []string{"organization"})
	req := resource.UpdateRequest{State: stateWith(t, sch, from), Plan: planWith(t, sch, to)}
	resp := &resource.UpdateResponse{State: stateWith(t, sch, from)}
	r.Update(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("personal + organization must be a diagnostic")
	}
	if len(calls) != 0 {
		t.Errorf("must not call select, got %#v", calls)
	}
}

func TestCreate_LookupError(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader(`{"error":"boom"}`)),
			Request:    req,
		}
	})
	sch := resourceSchema(t)
	m := model(t, "fw", []string{"user"})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), resource.CreateRequest{Plan: planWith(t, sch, m)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("select GET 500 must be a diagnostic")
	}
}

func TestCreate_FrameworkNotFound(t *testing.T) {
	r := stubResource(selectStub(t, map[string]map[string]interface{}{}, nil, nil))
	sch := resourceSchema(t)
	m := model(t, "missing", []string{"user"})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), resource.CreateRequest{Plan: planWith(t, sch, m)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("unknown framework must be a diagnostic")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Detail(), "missing") || strings.Contains(d.Summary(), "not found") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected not-found diagnostic, got %v", resp.Diagnostics)
	}
}

func TestCreate_ApplyScopeDiffError(t *testing.T) {
	entries := map[string]map[string]interface{}{
		"fw": {
			"id": "fw", "active": false, "selection_scopes": []string{},
			"display_name": "FW", "custom": false, "is_ready": true,
		},
	}
	r := stubResource(selectStub(t, entries, nil, func(req *http.Request) int {
		if req.Method == "POST" {
			return 500
		}
		return 200
	}))
	sch := resourceSchema(t)
	m := model(t, "fw", []string{"user"})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), resource.CreateRequest{Plan: planWith(t, sch, m)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("select POST 500 must be a diagnostic")
	}
}

func TestCreate_RefreshDisappeared(t *testing.T) {
	gets := 0
	r := stubResource(func(req *http.Request) *http.Response {
		if req.Method == "GET" {
			gets++
			if gets == 1 {
				raw, _ := json.Marshal(map[string]map[string]interface{}{
					"fw": {"id": "fw", "active": false, "selection_scopes": []string{}, "display_name": "FW", "custom": false, "is_ready": true},
				})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Request: req}
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}
	})
	sch := resourceSchema(t)
	m := model(t, "fw", []string{"user"})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), resource.CreateRequest{Plan: planWith(t, sch, m)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("refresh miss after write must be a diagnostic")
	}
}

func TestCreate_RefreshLookupError(t *testing.T) {
	gets := 0
	r := stubResource(func(req *http.Request) *http.Response {
		if req.Method == "GET" {
			gets++
			if gets == 1 {
				raw, _ := json.Marshal(map[string]map[string]interface{}{
					"fw": {"id": "fw", "active": false, "selection_scopes": []string{}, "display_name": "FW", "custom": false, "is_ready": true},
				})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Request: req}
			}
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(`{"error":"boom"}`)), Request: req}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}
	})
	sch := resourceSchema(t)
	m := model(t, "fw", []string{"user"})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), resource.CreateRequest{Plan: planWith(t, sch, m)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("refresh GET 500 must be a diagnostic")
	}
}

func TestSchema_EmptyScopesAllowed(t *testing.T) {
	sch := resourceSchema(t)
	scopes, ok := sch.Attributes["scopes"].(schema.SetAttribute)
	if !ok || !scopes.Required {
		t.Fatalf("scopes must be Required set, got %#v", sch.Attributes["scopes"])
	}
	// SizeAtLeast(1) would reject scopes = []; only OneOf on members is allowed.
	if len(scopes.Validators) != 1 {
		t.Errorf("scopes should only validate member values, got %d validators", len(scopes.Validators))
	}
}

func TestMetadataTypeName(t *testing.T) {
	r := &complianceFrameworkSelectionResource{}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_compliance_framework_selection" {
		t.Errorf("type name = %s", resp.TypeName)
	}
}

func TestModelCoversSchema(t *testing.T) {
	attrs := testutils.ResourceSchemaAttrs(t, &complianceFrameworkSelectionResource{})
	tags := testutils.TfsdkTags(resourceModel{})
	for name := range attrs {
		if _, ok := tags[name]; !ok {
			t.Errorf("missing tfsdk tag for %q", name)
		}
	}
}
