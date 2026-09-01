package crown_jewel

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stubDataSource(fn testutils.RoundTripFunc) *crownJewelDataSource {
	return &crownJewelDataSource{apiClient: testutils.NewStubAPIClient(fn)}
}

func dataSourceSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &datasource.SchemaResponse{}
	(&crownJewelDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema build failed: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// dataConfigWith builds a read-only tfsdk.Config from model. tfsdk.Config has no
// Set, so populate a State then copy its Raw (same pattern as servicenow tests).
func dataConfigWith(t *testing.T, sch schema.Schema, model stateModel) tfsdk.Config {
	t.Helper()
	st := tfsdk.State{Schema: sch}
	if diags := st.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("failed to seed config: %v", diags)
	}
	return tfsdk.Config{Schema: sch, Raw: st.Raw}
}

func TestDataSourceSchemaContracts(t *testing.T) {
	attrs := dataSourceSchema(t).Attributes

	gid, ok := attrs["group_unique_id"].(schema.StringAttribute)
	if !ok || !gid.Required {
		t.Errorf("group_unique_id must be Required, got %#v", attrs["group_unique_id"])
	}

	desc, ok := attrs["description"].(schema.StringAttribute)
	if !ok || !desc.Computed || desc.Required || desc.Optional {
		t.Errorf("description must be Computed-only, got %#v", attrs["description"])
	}

	id, ok := attrs["id"].(schema.StringAttribute)
	if !ok || !id.Computed {
		t.Errorf("id must be Computed, got %#v", attrs["id"])
	}
}

func TestDataSourceMetadataTypeName(t *testing.T) {
	d := &crownJewelDataSource{}
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_crown_jewel" {
		t.Errorf("unexpected type name: %s", resp.TypeName)
	}
}

// Read returns description for a user-marked jewel.
func TestDataSourceRead_Found(t *testing.T) {
	d := stubDataSource(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(
				`[{"group_unique_id":"vm_marked","description":"Customer data"}]`)),
			Request: req,
		}
	})
	sch := dataSourceSchema(t)
	cfg := stateModel{GroupUniqueID: types.StringValue("vm_marked")}
	req := datasource.ReadRequest{Config: dataConfigWith(t, sch, cfg)}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: sch}}
	d.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags: %v", resp.Diagnostics)
	}
	var out stateModel
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatalf("failed to read state: %v", diags)
	}
	if out.ID.ValueString() != "vm_marked" || out.Description.ValueString() != "Customer data" {
		t.Errorf("unexpected state: %+v", out)
	}
}

// Read errors when the asset is not user-marked.
func TestDataSourceRead_NotFound(t *testing.T) {
	d := stubDataSource(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`[]`)),
			Request:    req,
		}
	})
	sch := dataSourceSchema(t)
	cfg := stateModel{GroupUniqueID: types.StringValue("missing")}
	req := datasource.ReadRequest{Config: dataConfigWith(t, sch, cfg)}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: sch}}
	d.Read(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected not-found diagnostic")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "missing") {
		t.Errorf("diagnostic must name the group_unique_id, got: %v", resp.Diagnostics)
	}
}
