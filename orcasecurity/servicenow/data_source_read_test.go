package servicenow

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	fwdatasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dataSourceSchema drives the given data source's Schema method and returns the schema for
// seeding Config/State.
func dataSourceSchema(t *testing.T, ds datasource.DataSource) fwdatasourceschema.Schema {
	t.Helper()
	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema build failed: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// configWith builds a read-only tfsdk.Config populated from model. tfsdk.Config has no Set, so we
// seed a tfsdk.State (which does) and copy its Raw value — the two share the same tftypes shape.
func configWith(t *testing.T, sch fwdatasourceschema.Schema, model interface{}) tfsdk.Config {
	t.Helper()
	st := tfsdk.State{Schema: sch}
	if diags := st.Set(context.Background(), model); diags.HasError() {
		t.Fatalf("failed to seed config: %v", diags)
	}
	return tfsdk.Config{Schema: sch, Raw: st.Raw}
}

// readLookup runs a Read on the name-lookup data source against the given backend, configured to
// look up `name`, and returns the response for assertions.
func readLookup(t *testing.T, handler http.HandlerFunc, name string) *datasource.ReadResponse {
	t.Helper()
	sch := dataSourceSchema(t, NewServiceNowDataSource())
	cfg := configWith(t, sch, &itsmDataSourceModel{Name: types.StringValue(name)})
	ds := &itsmDataSource{apiClient: newTestClient(t, handler)}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: sch}}
	ds.Read(context.Background(), datasource.ReadRequest{Config: cfg}, resp)
	return resp
}

// readSchemaDS runs a Read on the schema data source against the given backend for the given
// resource id / schema type and returns the response for assertions.
func readSchemaDS(t *testing.T, handler http.HandlerFunc, snowType string) *datasource.ReadResponse {
	t.Helper()
	sch := dataSourceSchema(t, NewServiceNowSchemaDataSource())
	cfg := configWith(t, sch, &schemaDataSourceModel{
		ResourceID: types.StringValue("res-1"),
		Type:       types.StringValue(snowType),
		Fields:     []schemaFieldModel{},
		Elements:   types.ListNull(types.StringType),
	})
	ds := &schemaDataSource{apiClient: newTestClient(t, handler)}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: sch}}
	ds.Read(context.Background(), datasource.ReadRequest{Config: cfg}, resp)
	return resp
}

// Read on the lookup data source must find the resource by name and populate id/url/username.
func TestDataSourceRead_Success(t *testing.T) {
	// ListServiceNowITSMResources shape: {status, data:[...]}
	resp := readLookup(t, jsonOK(`{"status":"success","data":[{"id":"res-1","name":"prod","host_url":"https://acme.service-now.com","data":{"username":"svc-orca"}}]}`), "prod")
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags: %v", resp.Diagnostics)
	}
	var out itsmDataSourceModel
	resp.State.Get(context.Background(), &out)
	if out.ID.ValueString() != "res-1" {
		t.Errorf("id: got %q", out.ID.ValueString())
	}
	if out.ServiceNowURL.ValueString() != "https://acme.service-now.com" {
		t.Errorf("servicenow_url: got %q", out.ServiceNowURL.ValueString())
	}
	if out.Username.ValueString() != "svc-orca" {
		t.Errorf("username: got %q", out.Username.ValueString())
	}
}

// A name with no matching resource must surface a "not found" error diagnostic.
func TestDataSourceRead_NotFoundErrors(t *testing.T) {
	resp := readLookup(t, jsonOK(`{"status":"success","data":[]}`), "absent")
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a not-found error diagnostic")
	}
}

// An API failure during lookup must surface an error diagnostic.
func TestDataSourceRead_APIErrorSurfacesDiag(t *testing.T) {
	resp := readLookup(t, httpError(http.StatusInternalServerError, "boom"), "prod")
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic on API failure")
	}
}

// Read on the schema data source must translate every schema field into a schemaFieldModel and
// build the flat elements list.
func TestSchemaDataSourceRead_Success(t *testing.T) {
	resp := readSchemaDS(t, jsonOK(`{"status":"success","data":[
		{"element":"short_description","label":"Short description","type":"string","default_value":"","max_length":"160","choice":"0"},
		{"element":"urgency","label":"Urgency","type":"integer","default_value":"3","max_length":"40","choice":"3"}
	]}`), "itsm")
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags: %v", resp.Diagnostics)
	}
	var out schemaDataSourceModel
	resp.State.Get(context.Background(), &out)
	if len(out.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(out.Fields))
	}
	if out.Fields[0].Element.ValueString() != "short_description" {
		t.Errorf("first field element: got %q", out.Fields[0].Element.ValueString())
	}
	var elements []string
	out.Elements.ElementsAs(context.Background(), &elements, false)
	if len(elements) != 2 || elements[0] != "short_description" || elements[1] != "urgency" {
		t.Errorf("elements list mismatch: %v", elements)
	}
}

// An empty schema response must yield empty fields/elements, not an error.
func TestSchemaDataSourceRead_EmptyResult(t *testing.T) {
	resp := readSchemaDS(t, jsonOK(`{"status":"success","data":[]}`), "sir")
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags: %v", resp.Diagnostics)
	}
	var out schemaDataSourceModel
	resp.State.Get(context.Background(), &out)
	if len(out.Fields) != 0 {
		t.Errorf("expected 0 fields, got %d", len(out.Fields))
	}
}

// An API failure during schema read must surface an error diagnostic.
func TestSchemaDataSourceRead_APIErrorSurfacesDiag(t *testing.T) {
	resp := readSchemaDS(t, httpError(http.StatusInternalServerError, "boom"), "itsm")
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic on API failure")
	}
}
