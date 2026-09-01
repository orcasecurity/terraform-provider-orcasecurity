package crown_jewel

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stubResource(fn testutils.RoundTripFunc) *crownJewelResource {
	return &crownJewelResource{apiClient: testutils.NewStubAPIClient(fn)}
}

func nullTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectValueMust(
			map[string]attr.Type{
				"create": types.StringType,
				"update": types.StringType,
				"delete": types.StringType,
			},
			map[string]attr.Value{
				"create": types.StringNull(),
				"update": types.StringNull(),
				"delete": types.StringNull(),
			},
		),
	}
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

func stateWith(t *testing.T, sch schema.Schema, model resourceModel) tfsdk.State {
	t.Helper()
	model.Timeouts = nullTimeouts()
	st := tfsdk.State{Schema: sch}
	if diags := st.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("failed to seed state: %v", diags)
	}
	return st
}

func planWith(t *testing.T, sch schema.Schema, model resourceModel) tfsdk.Plan {
	t.Helper()
	model.Timeouts = nullTimeouts()
	p := tfsdk.Plan{Schema: sch}
	if diags := p.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("failed to seed plan: %v", diags)
	}
	return p
}

func TestSchemaContracts(t *testing.T) {
	sch := resourceSchema(t)
	if !strings.Contains(sch.Description, "user-marked") {
		t.Errorf("schema Description must mention already user-marked create, got %q", sch.Description)
	}

	attrs := sch.Attributes

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

	if _, ok := attrs["timeouts"]; !ok {
		t.Error("timeouts block must be present so create/update/delete HTTP timeouts are configurable")
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
	model := resourceModel{
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
		switch {
		case req.Method == "POST" && strings.Contains(req.URL.Path, "/api/serving-layer/query"):
			return jsonOK(req, `{"status":"success","data":[{"group_unique_id":"tf-miss","data":{"GroupUniqueId":{"value":"tf-miss"}}}]}`)
		case req.Method == "POST":
			return jsonOK(req, `{"status":"success"}`)
		case req.Method == "GET" && strings.Contains(req.URL.Path, "/api/attack_paths/crown_jewels"):
			return jsonOK(req, `[]`)
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil
		}
	})
	sch := resourceSchema(t)
	plan := resourceModel{
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
	var jewelsGets int
	r := stubResource(func(req *http.Request) *http.Response {
		switch {
		case req.Method == "POST" && strings.Contains(req.URL.Path, "/api/serving-layer/query"):
			return jsonOK(req, `{"status":"success","data":[{"group_unique_id":"tf-keep","data":{"GroupUniqueId":{"value":"tf-keep"}}}]}`)
		case req.Method == "POST":
			return jsonOK(req, `{"status":"success"}`)
		case req.Method == "GET" && strings.Contains(req.URL.Path, "/api/attack_paths/crown_jewels"):
			jewelsGets++
			if jewelsGets == 1 {
				return jsonOK(req, `[]`)
			}
			return jsonOK(req, `[{"group_unique_id":"tf-keep","description":"API NORMALIZED DIFFERENT"}]`)
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil
		}
	})
	sch := resourceSchema(t)
	plan := resourceModel{
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

	var out resourceModel
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

func jsonOK(req *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

// Create matches the UI: Mark is not offered when the asset is already user-marked.
func TestCreate_AlreadyMarkedFails(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		if req.Method == "POST" {
			t.Fatal("must not POST when the asset is already user-marked")
		}
		if req.Method == "GET" && strings.Contains(req.URL.Path, "/api/attack_paths/crown_jewels") {
			return jsonOK(req, `[{"group_unique_id":"vm_marked","description":"Customer data"}]`)
		}
		t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
		return nil
	})
	sch := resourceSchema(t)
	plan := resourceModel{
		ID:            types.StringUnknown(),
		GroupUniqueID: types.StringValue("vm_marked"),
		Description:   types.StringValue("Customer data"),
	}
	req := resource.CreateRequest{Plan: planWith(t, sch, plan)}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected create to fail when the asset is already user-marked")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "vm_marked") {
		t.Errorf("diagnostic must name the group_unique_id, got: %v", resp.Diagnostics)
	}
	if !strings.Contains(strings.ToLower(detail), "import") {
		t.Errorf("diagnostic must tell the user to import, got: %v", resp.Diagnostics)
	}
}

// Create must not POST a phantom CrownJewel row for an id that is not in inventory.
func TestCreate_UnknownAssetFails(t *testing.T) {
	r := stubResource(func(req *http.Request) *http.Response {
		if req.Method == "POST" && strings.Contains(req.URL.Path, "/api/attack_paths/crown_jewels") {
			t.Fatal("must not POST when the asset is not in inventory")
		}
		switch {
		case req.Method == "GET" && strings.Contains(req.URL.Path, "/api/attack_paths/crown_jewels"):
			return jsonOK(req, `[]`)
		case req.Method == "POST" && strings.Contains(req.URL.Path, "/api/serving-layer/query"):
			return jsonOK(req, `{"status":"success","data":[]}`)
		default:
			t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
			return nil
		}
	})
	sch := resourceSchema(t)
	plan := resourceModel{
		ID:            types.StringUnknown(),
		GroupUniqueID: types.StringValue("tf-phantom"),
		Description:   types.StringValue("Customer data"),
	}
	req := resource.CreateRequest{Plan: planWith(t, sch, plan)}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected create to fail when group_unique_id is not in inventory")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "tf-phantom") {
		t.Errorf("diagnostic must name the group_unique_id, got: %v", resp.Diagnostics)
	}
}

// Create on an Orca-detected (but not user-marked) asset still POSTs — the UI
// allows Mark there and the API upserts a user overlay (hybrid).
func TestCreate_OrcaDetectedSucceeds(t *testing.T) {
	var posted bool
	r := stubResource(func(req *http.Request) *http.Response {
		switch {
		case req.Method == "GET" && strings.Contains(req.URL.Path, "/api/attack_paths/crown_jewels"):
			if posted {
				return jsonOK(req, `[{"group_unique_id":"vm_orca","description":"Data: Financial information"}]`)
			}
			return jsonOK(req, `[]`)
		case req.Method == "POST" && strings.Contains(req.URL.Path, "/api/serving-layer/query"):
			return jsonOK(req, `{
				"status":"success",
				"data":[{
					"group_unique_id":"vm_orca",
					"data":{"GroupUniqueId":{"value":"vm_orca"}}
				}]
			}`)
		case req.Method == "POST" && strings.Contains(req.URL.Path, "/api/attack_paths/crown_jewels"):
			posted = true
			return jsonOK(req, `{"status":"success"}`)
		default:
			t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
			return nil
		}
	})
	sch := resourceSchema(t)
	plan := resourceModel{
		ID:            types.StringUnknown(),
		GroupUniqueID: types.StringValue("vm_orca"),
		Description:   types.StringValue("Data: Financial information"),
	}
	req := resource.CreateRequest{Plan: planWith(t, sch, plan)}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Orca-detected unmarked asset must be markable, got: %v", resp.Diagnostics)
	}
	if !posted {
		t.Fatal("expected POST to mark the Orca-detected asset")
	}
}
