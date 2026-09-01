package compliance_framework_selection

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

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

func model(t *testing.T, frameworkID string, scopes []string, restore bool, original []string) resourceModel {
	t.Helper()
	return resourceModel{
		ID:               types.StringValue(frameworkID),
		FrameworkID:      types.StringValue(frameworkID),
		Scopes:           testutils.StringSet(t, scopes...),
		RestoreOnDestroy: types.BoolValue(restore),
		OriginalScopes:   testutils.StringSet(t, original...),
		Active:           types.BoolValue(len(scopes) > 0),
		DisplayName:      types.StringValue("Lab"),
		Custom:           types.BoolValue(false),
		IsReady:          types.BoolValue(true),
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
	m := model(t, "gone", []string{"user"}, false, nil)
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
	m := model(t, "cost_optimization", []string{"user"}, false, []string{})
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
	if out.Active.ValueBool() {
		t.Error("active must be false when scopes are empty")
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
			from := model(t, "fw", tt.from, false, nil)
			to := model(t, "fw", tt.to, false, nil)
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
	m := model(t, "fw", []string{"user"}, false, []string{})
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

func TestDelete_RestoreOnDestroy(t *testing.T) {
	t.Run("empty original deselects all", func(t *testing.T) {
		var calls []httpCall
		r := stubResource(selectStub(t, map[string]map[string]interface{}{}, &calls, nil))
		sch := resourceSchema(t)
		m := model(t, "fw", []string{"user", "organization"}, true, []string{})
		req := resource.DeleteRequest{State: stateWith(t, sch, m)}
		resp := &resource.DeleteResponse{}
		r.Delete(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("delete: %v", resp.Diagnostics)
		}
		got := map[string]string{}
		for _, c := range calls {
			got[c.Scope] = c.Method
		}
		if got["user"] != "DELETE" || got["organization"] != "DELETE" {
			t.Errorf("calls = %#v, want DELETE of user and organization", calls)
		}
	})
	t.Run("organization to user swaps", func(t *testing.T) {
		var calls []httpCall
		r := stubResource(selectStub(t, map[string]map[string]interface{}{}, &calls, nil))
		sch := resourceSchema(t)
		m := model(t, "fw", []string{"user"}, true, []string{"organization"})
		req := resource.DeleteRequest{State: stateWith(t, sch, m)}
		resp := &resource.DeleteResponse{}
		r.Delete(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("delete: %v", resp.Diagnostics)
		}
		want := map[string]string{"organization": "POST", "user": "DELETE"}
		got := map[string]string{}
		for _, c := range calls {
			got[c.Scope] = c.Method
		}
		if got["organization"] != want["organization"] || got["user"] != want["user"] {
			t.Errorf("calls = %#v, want POST organization + DELETE user", calls)
		}
	})
	t.Run("swallows 404", func(t *testing.T) {
		r := stubResource(selectStub(t, map[string]map[string]interface{}{}, nil, func(*http.Request) int { return 404 }))
		sch := resourceSchema(t)
		m := model(t, "fw", []string{"user"}, true, []string{})
		req := resource.DeleteRequest{State: stateWith(t, sch, m)}
		resp := &resource.DeleteResponse{}
		r.Delete(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("404 on restore must be swallowed, got %v", resp.Diagnostics)
		}
	})
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
