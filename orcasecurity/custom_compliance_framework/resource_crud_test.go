package custom_compliance_framework

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stubResource(fn testutils.RoundTripFunc) *customComplianceFrameworkResource {
	return &customComplianceFrameworkResource{apiClient: testutils.NewStubAPIClient(fn)}
}

func resourceSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &resource.SchemaResponse{}
	(&customComplianceFrameworkResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func stateWith(t *testing.T, sch schema.Schema, model customComplianceFrameworkResourceModel) tfsdk.State {
	t.Helper()
	st := tfsdk.State{Schema: sch}
	if diags := st.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	return st
}

func planWith(t *testing.T, sch schema.Schema, model customComplianceFrameworkResourceModel) tfsdk.Plan {
	t.Helper()
	p := tfsdk.Plan{Schema: sch}
	if diags := p.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("plan: %v", diags)
	}
	return p
}

const fwJSON = `{"data":{"id":"3887","display_name":"Lab","custom":true,"active":false,"selection_scopes":[],"visibility":"Personal"}}`

const catalogJSON = `{"data":{"frameworks":[{"framework_id":"3887","name":"Lab","display_name":"Lab","custom":true,"sections":[{"id":"1","name":"Flat","tests":[{"rule_id":"r1","reference_id":"1.1","priority":"Medium"}]}]}]}}`

func happyCRUD(req *http.Request) *http.Response {
	switch {
	case req.Method == "POST" && req.URL.Path == "/api/compliance/frameworks":
		return testutils.JSONResponse(req, 200, `{"data":{"id":3887,"name":"Lab","description":""}}`)
	case req.Method == "PUT" && strings.HasPrefix(req.URL.Path, "/api/compliance/frameworks/"):
		return testutils.JSONResponse(req, 200, `{"data":{"id":3887,"name":"Lab","description":""}}`)
	case req.Method == "GET" && req.URL.Path == "/api/compliance/frameworks/3887":
		return testutils.JSONResponse(req, 200, fwJSON)
	case req.Method == "GET" && req.URL.Path == "/api/compliance/catalog/3887":
		return testutils.JSONResponse(req, 200, catalogJSON)
	case req.Method == "DELETE" && req.URL.Path == "/api/compliance/frameworks/3887":
		return testutils.JSONResponse(req, 200, `{}`)
	default:
		return testutils.JSONResponse(req, 500, `{"error":"unexpected `+req.Method+` `+req.URL.Path+`"}`)
	}
}

func planModel(t *testing.T) customComplianceFrameworkResourceModel {
	t.Helper()
	return customComplianceFrameworkResourceModel{
		Name:               types.StringValue("Lab"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           oneSection(t),
	}
}

func stateModel(t *testing.T) customComplianceFrameworkResourceModel {
	t.Helper()
	m := planModel(t)
	m.ID = types.StringValue("3887")
	return m
}

func TestRead_NotFoundRemovesResource(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		return testutils.JSONResponse(req, 404, `{"error":"Framework 3887 not found."}`)
	})
	sch := resourceSchema(t)
	m := stateModel(t)
	req := resource.ReadRequest{State: stateWith(t, sch, m)}
	resp := &resource.ReadResponse{State: stateWith(t, sch, m)}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("404 must RemoveResource, got %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("expected RemoveResource, got %v", resp.State.Raw)
	}
}

func TestRead_PopulateMapsCatalog(t *testing.T) {
	r := stubResource(happyCRUD)
	sch := resourceSchema(t)
	m := stateModel(t)
	req := resource.ReadRequest{State: stateWith(t, sch, m)}
	resp := &resource.ReadResponse{State: stateWith(t, sch, m)}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var out customComplianceFrameworkResourceModel
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	if out.Name.ValueString() != "Lab" {
		t.Errorf("name: %q", out.Name.ValueString())
	}
	if out.Visibility.ValueString() != "Personal" {
		t.Errorf("visibility: %q", out.Visibility.ValueString())
	}
	if len(out.Sections.Elements()) != 1 {
		t.Fatalf("sections: %#v", out.Sections)
	}
	sec := out.Sections.Elements()[0].(types.Object)
	if sec.Attributes()["name"].(types.String).ValueString() != "Flat" {
		t.Errorf("section name: %#v", sec.Attributes()["name"])
	}
	if sec.Attributes()["section_id_in_framework"].(types.String).ValueString() != "1" {
		t.Errorf("section_id_in_framework: %#v", sec.Attributes()["section_id_in_framework"])
	}
	tests := sec.Attributes()["tests"].(types.List)
	if len(tests.Elements()) != 1 {
		t.Fatalf("tests: %#v", tests)
	}
	test0 := tests.Elements()[0].(types.Object)
	if test0.Attributes()["rule_id"].(types.String).ValueString() != "r1" {
		t.Errorf("rule_id: %#v", test0.Attributes()["rule_id"])
	}
	if test0.Attributes()["rule_id_in_framework"].(types.String).ValueString() != "1.1" {
		t.Errorf("rule_id_in_framework: %#v", test0.Attributes()["rule_id_in_framework"])
	}
}

func TestCreate_APIErrorSurfaces(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		if req.Method == "POST" {
			return testutils.JSONResponse(req, 400, `{"error":"name already exists"}`)
		}
		t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
		return nil
	})
	sch := resourceSchema(t)
	req := resource.CreateRequest{Plan: planWith(t, sch, planModel(t))}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("API create error must surface as a diagnostic")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Detail(), "name already exists") || strings.Contains(d.Summary(), "creating") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected create error diagnostic, got %v", resp.Diagnostics)
	}
}

func TestCreate_RefreshMapsCatalog(t *testing.T) {
	r := stubResource(happyCRUD)
	sch := resourceSchema(t)
	req := resource.CreateRequest{Plan: planWith(t, sch, planModel(t))}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	var out customComplianceFrameworkResourceModel
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	if out.ID.ValueString() != "3887" {
		t.Errorf("id: %q", out.ID.ValueString())
	}
	if len(out.Sections.Elements()) != 1 {
		t.Fatalf("sections after create: %#v", out.Sections)
	}
}

func TestUpdate_APIErrorSurfaces(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		if req.Method == "PUT" {
			return testutils.JSONResponse(req, 500, `{"error":"boom"}`)
		}
		t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
		return nil
	})
	sch := resourceSchema(t)
	m := stateModel(t)
	req := resource.UpdateRequest{Plan: planWith(t, sch, m), State: stateWith(t, sch, m)}
	resp := &resource.UpdateResponse{State: stateWith(t, sch, m)}
	r.Update(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("API update error must surface")
	}
}

func TestUpdate_RefreshMapsCatalog(t *testing.T) {
	r := stubResource(happyCRUD)
	sch := resourceSchema(t)
	m := stateModel(t)
	req := resource.UpdateRequest{Plan: planWith(t, sch, m), State: stateWith(t, sch, m)}
	resp := &resource.UpdateResponse{State: stateWith(t, sch, m)}
	r.Update(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}
	var out customComplianceFrameworkResourceModel
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	if out.Name.ValueString() != "Lab" {
		t.Errorf("name: %q", out.Name.ValueString())
	}
}

func TestDelete_404Ignored(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		if req.Method != "DELETE" {
			t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
		}
		return testutils.JSONResponse(req, 404, `{"error":"Framework 3887 not found."}`)
	})
	sch := resourceSchema(t)
	req := resource.DeleteRequest{State: stateWith(t, sch, stateModel(t))}
	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("404 on delete must be ignored, got %v", resp.Diagnostics)
	}
}

func TestDelete_OtherErrorSurfaced(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		return testutils.JSONResponse(req, 500, `{"error":"boom"}`)
	})
	sch := resourceSchema(t)
	req := resource.DeleteRequest{State: stateWith(t, sch, stateModel(t))}
	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("non-404 delete error must surface")
	}
}

func TestPopulate_CatalogMissing(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		if strings.Contains(req.URL.Path, "/catalog/") {
			return testutils.JSONResponse(req, 200, `{"data":{"frameworks":[]}}`)
		}
		return testutils.JSONResponse(req, 200, fwJSON)
	})
	m := stateModel(t)
	ok, d := r.populate(context.Background(), &m)
	if ok {
		t.Fatal("empty catalog must not succeed")
	}
	if !d.HasError() {
		t.Fatal("empty catalog must be a diagnostic")
	}
}

func TestMetadataTypeName(t *testing.T) {
	r := &customComplianceFrameworkResource{}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_custom_compliance_framework" {
		t.Errorf("type name = %s", resp.TypeName)
	}
}
