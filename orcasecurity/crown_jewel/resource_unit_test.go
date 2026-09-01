package crown_jewel

import (
	"context"
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

func stubResource(fn testutils.RoundTripFunc) *crownJewelResource {
	return &crownJewelResource{apiClient: testutils.NewStubAPIClient(fn)}
}

func resourceSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &resource.SchemaResponse{}
	(&crownJewelResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema build failed: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func stateWith(t *testing.T, sch schema.Schema, model stateModel) tfsdk.State {
	t.Helper()
	st := tfsdk.State{Schema: sch}
	if diags := st.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("failed to seed state: %v", diags)
	}
	return st
}

func planWith(t *testing.T, sch schema.Schema, model stateModel) tfsdk.Plan {
	t.Helper()
	p := tfsdk.Plan{Schema: sch}
	if diags := p.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("failed to seed plan: %v", diags)
	}
	return p
}

func TestSchemaContracts(t *testing.T) {
	attrs := resourceSchema(t).Attributes

	gid, ok := attrs["group_unique_id"].(schema.StringAttribute)
	if !ok || !gid.Required || len(gid.PlanModifiers) != 1 {
		t.Errorf("group_unique_id must be Required with RequiresReplace, got %#v", attrs["group_unique_id"])
	}

	desc, ok := attrs["description"].(schema.StringAttribute)
	if !ok || !desc.Required || desc.Optional || len(desc.Validators) < 2 {
		t.Errorf("description must be Required with length+non-whitespace validators, got %#v", attrs["description"])
	}

	id, ok := attrs["id"].(schema.StringAttribute)
	if !ok || !id.Computed {
		t.Errorf("id must be Computed, got %#v", attrs["id"])
	}
}

func TestMetadataTypeName(t *testing.T) {
	r := &crownJewelResource{}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_crown_jewel" {
		t.Errorf("unexpected type name: %s", resp.TypeName)
	}
}

// Read with a nil lookup must RemoveResource, not surface a spurious error.
func TestRead_NotFoundRemovesResource(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`[]`)),
			Request:    req,
		}
	})
	sch := resourceSchema(t)
	model := stateModel{
		ID:            types.StringValue("gone"),
		GroupUniqueID: types.StringValue("gone"),
		Description:   types.StringValue("Customer data"),
	}
	req := resource.ReadRequest{State: stateWith(t, sch, model)}
	resp := &resource.ReadResponse{State: stateWith(t, sch, model)}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("missing remote jewel must not error, got: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("expected RemoveResource (null raw state), got %v", resp.State.Raw)
	}
}

// Create must fail cleanly when POST succeeds but the follow-up GET misses the jewel.
func TestCreate_RefetchMissSurfacesDiag(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		switch req.Method {
		case "POST":
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"status":"success"}`)),
				Request:    req,
			}
		case "GET":
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Request:    req,
			}
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil
		}
	})
	sch := resourceSchema(t)
	plan := stateModel{
		ID:            types.StringUnknown(),
		GroupUniqueID: types.StringValue("tf-miss"),
		Description:   types.StringValue("Customer data"),
	}
	req := resource.CreateRequest{Plan: planWith(t, sch, plan)}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected create to surface refetch-miss as a diagnostic")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "tf-miss") {
		t.Errorf("diagnostic must name the group_unique_id, got: %v", resp.Diagnostics)
	}
}

// Create success keeps Required attributes from the plan — including description —
// even when the API refetch returns a different string (normalization must not
// rewrite Required fields and cause "inconsistent result after apply").
func TestCreate_SuccessKeepsPlanRequiredAttrs(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		switch req.Method {
		case "POST":
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"status":"success"}`)),
				Request:    req,
			}
		case "GET":
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(strings.NewReader(
					`[{"group_unique_id":"tf-keep","description":"API NORMALIZED DIFFERENT"}]`)),
				Request: req,
			}
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil
		}
	})
	sch := resourceSchema(t)
	plan := stateModel{
		ID:            types.StringUnknown(),
		GroupUniqueID: types.StringValue("tf-keep"),
		Description:   types.StringValue("Customer data"),
	}
	req := resource.CreateRequest{Plan: planWith(t, sch, plan)}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags: %v", resp.Diagnostics)
	}

	var out stateModel
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatalf("failed to read state: %v", diags)
	}
	if out.GroupUniqueID.ValueString() != "tf-keep" {
		t.Errorf("group_unique_id must stay as planned, got %q", out.GroupUniqueID.ValueString())
	}
	if out.ID.ValueString() != "tf-keep" {
		t.Errorf("id must equal planned group_unique_id, got %q", out.ID.ValueString())
	}
	if out.Description.ValueString() != "Customer data" {
		t.Errorf("description must stay as planned (not API), got %q", out.Description.ValueString())
	}
}
