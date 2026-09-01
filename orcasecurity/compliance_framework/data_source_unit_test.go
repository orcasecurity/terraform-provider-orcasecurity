package compliance_framework

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func jsonResp(req *http.Request, code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func singleSchema(t *testing.T) tfsdk.State {
	t.Helper()
	d := &complianceFrameworkDataSource{}
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)
	return tfsdk.State{Schema: schemaResp.Schema}
}

func singleConfig(t *testing.T, id string) tfsdk.Config {
	t.Helper()
	st := singleSchema(t)
	cfgModel := singleDataSourceModel{frameworkModel: frameworkModel{
		ID:                    types.StringValue(id),
		SelectionScopes:       types.ListNull(types.StringType),
		FrameworkCloudVendors: types.ListNull(types.StringType),
	}, Sections: types.ListNull(catalogSectionObjectType(maxCatalogDepth - 1))}
	if diags := st.Set(context.Background(), &cfgModel); diags.HasError() {
		t.Fatalf("set: %v", diags)
	}
	return tfsdk.Config{Schema: st.Schema, Raw: st.Raw}
}

func listSchema(t *testing.T) tfsdk.State {
	t.Helper()
	d := &complianceFrameworksDataSource{}
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)
	return tfsdk.State{Schema: schemaResp.Schema}
}

func listConfig(t *testing.T, model frameworksDataSourceModel) tfsdk.Config {
	t.Helper()
	st := listSchema(t)
	if diags := st.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set: %v", diags)
	}
	return tfsdk.Config{Schema: st.Schema, Raw: st.Raw}
}

const selectJSON = `{
	"cost_optimization": {
		"id": "cost_optimization",
		"active": false,
		"selection_scopes": [],
		"display_name": "Cost Optimization",
		"custom": false,
		"type": "Orca Frameworks",
		"description": "cost"
	},
	"lab": {
		"id": "lab",
		"active": true,
		"selection_scopes": ["user"],
		"display_name": "Lab Custom",
		"custom": true,
		"visibility": "Personal"
	}
}`

func TestListDataSource_FilterMatrix(t *testing.T) {
	d := &complianceFrameworksDataSource{apiClient: testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		if req.URL.Path != "/api/compliance/frameworks/select" {
			t.Fatalf("path %s", req.URL.Path)
		}
		return jsonResp(req, 200, selectJSON)
	})}
	cfg := listConfig(t, frameworksDataSourceModel{
		Custom:      types.BoolValue(true),
		Active:      types.BoolNull(),
		Type:        types.StringNull(),
		DisplayName: types.StringNull(),
		Search:      types.StringNull(),
	})
	resp := &datasource.ReadResponse{State: listSchema(t)}
	d.Read(context.Background(), datasource.ReadRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var out frameworksDataSourceModel
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	if len(out.Frameworks) != 1 || out.Frameworks[0].ID.ValueString() != "lab" {
		t.Fatalf("custom filter: %+v", out.Frameworks)
	}

	cfg = listConfig(t, frameworksDataSourceModel{
		Custom:      types.BoolNull(),
		Active:      types.BoolValue(false),
		Type:        types.StringValue("Orca Frameworks"),
		DisplayName: types.StringNull(),
		Search:      types.StringNull(),
	})
	resp = &datasource.ReadResponse{State: listSchema(t)}
	d.Read(context.Background(), datasource.ReadRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("type+inactive: %v", resp.Diagnostics)
	}
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	if len(out.Frameworks) != 1 || out.Frameworks[0].ID.ValueString() != "cost_optimization" {
		t.Fatalf("type+inactive: %+v", out.Frameworks)
	}

	cfg = listConfig(t, frameworksDataSourceModel{
		Custom:      types.BoolNull(),
		Active:      types.BoolNull(),
		Type:        types.StringNull(),
		DisplayName: types.StringNull(),
		Search:      types.StringValue("LAB"),
	})
	resp = &datasource.ReadResponse{State: listSchema(t)}
	d.Read(context.Background(), datasource.ReadRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("search: %v", resp.Diagnostics)
	}
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	if len(out.Frameworks) != 1 || out.Frameworks[0].DisplayName.ValueString() != "Lab Custom" {
		t.Fatalf("search: %+v", out.Frameworks)
	}
}

func TestListDataSource_APIError(t *testing.T) {
	d := &complianceFrameworksDataSource{apiClient: testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		return jsonResp(req, 500, `{"error":"boom"}`)
	})}
	cfg := listConfig(t, frameworksDataSourceModel{
		Custom:      types.BoolNull(),
		Active:      types.BoolNull(),
		Type:        types.StringNull(),
		DisplayName: types.StringNull(),
		Search:      types.StringNull(),
	})
	resp := &datasource.ReadResponse{State: listSchema(t)}
	d.Read(context.Background(), datasource.ReadRequest{Config: cfg}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("select 500 must be a diagnostic")
	}
}

func TestSingleDataSource_MapsSectionIDAndCISLevel(t *testing.T) {
	d := &complianceFrameworkDataSource{apiClient: testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		switch req.URL.Path {
		case "/api/compliance/frameworks/gcp_cis_3.0.0":
			return jsonResp(req, 200, `{"data":{"id":"gcp_cis_3.0.0","display_name":"GCP CIS","custom":false,"active":true,"selection_scopes":["user"],"type":"CIS"}}`)
		case "/api/compliance/catalog/gcp_cis_3.0.0":
			return jsonResp(req, 200, `{"data":{"frameworks":[{"framework_id":"gcp_cis_3.0.0","name":"GCP CIS","sections":[{"id":"1","name":"Identity","tests":[{"rule_id":"r1","reference_id":"1.1","cis_level":["Level 1"]}]}]}]}}`)
		default:
			t.Fatalf("path %s", req.URL.Path)
			return nil
		}
	})}
	resp := &datasource.ReadResponse{State: singleSchema(t)}
	d.Read(context.Background(), datasource.ReadRequest{Config: singleConfig(t, "gcp_cis_3.0.0")}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var out singleDataSourceModel
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	if out.ID.ValueString() != "gcp_cis_3.0.0" || out.DisplayName.ValueString() != "GCP CIS" {
		t.Errorf("framework: id=%s name=%s", out.ID, out.DisplayName)
	}
	if len(out.Sections.Elements()) != 1 {
		t.Fatalf("sections: %#v", out.Sections)
	}
	sec := out.Sections.Elements()[0].(types.Object)
	if sec.Attributes()["id"].(types.String).ValueString() != "1" {
		t.Errorf("section id: %#v", sec.Attributes()["id"])
	}
	test0 := sec.Attributes()["tests"].(types.List).Elements()[0].(types.Object)
	levels := test0.Attributes()["cis_level"].(types.List)
	if levels.IsNull() || len(levels.Elements()) != 1 || levels.Elements()[0].(types.String).ValueString() != "Level 1" {
		t.Errorf("cis_level: %#v", test0.Attributes()["cis_level"])
	}
}

func TestSingleDataSource_CatalogMissing(t *testing.T) {
	d := &complianceFrameworkDataSource{apiClient: testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		if strings.Contains(req.URL.Path, "/catalog/") {
			return jsonResp(req, 200, `{"data":{"frameworks":[]}}`)
		}
		return jsonResp(req, 200, `{"data":{"id":"x","display_name":"X","custom":true,"active":false,"selection_scopes":[]}}`)
	})}
	resp := &datasource.ReadResponse{State: singleSchema(t)}
	d.Read(context.Background(), datasource.ReadRequest{Config: singleConfig(t, "x")}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("empty catalog must be a diagnostic")
	}
}

func TestSingleDataSource_NotFound(t *testing.T) {
	d := &complianceFrameworkDataSource{apiClient: testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"error":"Framework missing not found."}`)),
			Request:    req,
		}
	})}
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)
	cfgModel := singleDataSourceModel{frameworkModel: frameworkModel{
		ID:                    types.StringValue("missing"),
		SelectionScopes:       types.ListNull(types.StringType),
		FrameworkCloudVendors: types.ListNull(types.StringType),
	}, Sections: types.ListNull(catalogSectionObjectType(maxCatalogDepth - 1))}
	st := tfsdk.State{Schema: schemaResp.Schema}
	if diags := st.Set(context.Background(), &cfgModel); diags.HasError() {
		t.Fatalf("set: %v", diags)
	}
	cfg := tfsdk.Config{Schema: schemaResp.Schema, Raw: st.Raw}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: cfg}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("missing framework must be a diagnostic, not empty state")
	}
	found := false
	for _, diag := range resp.Diagnostics {
		if strings.Contains(diag.Detail(), "missing") || strings.Contains(diag.Summary(), "not found") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected not-found diagnostic, got %v", resp.Diagnostics)
	}
}
