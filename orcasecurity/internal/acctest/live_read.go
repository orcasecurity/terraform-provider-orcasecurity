package acctest

import (
	"context"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type LiveAttrCheck struct {
	Path path.Path
	// Empty Want: assert only that Read populated the attr.
	Want string
}

// ImportState+Read against the live API; no apply/destroy/state mutation.
func RunLiveImportRead(t *testing.T, r resource.Resource, importID string, checks ...LiveAttrCheck) {
	t.Helper()
	client := APIClient(t)
	ctx := context.Background()

	schema := liveSchema(ctx, t, r)
	configureLive(ctx, t, r, client)
	state := liveImportState(ctx, t, r, schema, importID)

	readResp := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &readResp)
	failOnDiags(t, "read", readResp.Diagnostics)

	assertLiveAttrs(ctx, t, readResp.State, checks)
}

func failOnDiags(t *testing.T, step string, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() {
		t.Fatalf("%s: %v", step, diags)
	}
}

func liveSchema(ctx context.Context, t *testing.T, r resource.Resource) rschema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &resp)
	failOnDiags(t, "schema", resp.Diagnostics)
	return resp.Schema
}

func configureLive(ctx context.Context, t *testing.T, r resource.Resource, client *api_client.APIClient) {
	t.Helper()
	configurable, ok := r.(resource.ResourceWithConfigure)
	if !ok {
		t.Fatal("resource does not implement ResourceWithConfigure")
	}
	var resp resource.ConfigureResponse
	configurable.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &resp)
	failOnDiags(t, "configure", resp.Diagnostics)
}

// Exercise real ImportState parsing, not hand-built state.
func liveImportState(ctx context.Context, t *testing.T, r resource.Resource, schema rschema.Schema, importID string) tfsdk.State {
	t.Helper()
	importable, ok := r.(resource.ResourceWithImportState)
	if !ok {
		t.Fatal("resource does not implement ResourceWithImportState")
	}
	resp := resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schema,
			Raw:    tftypes.NewValue(schema.Type().TerraformType(ctx), nil),
		},
	}
	importable.ImportState(ctx, resource.ImportStateRequest{ID: importID}, &resp)
	failOnDiags(t, "import state", resp.Diagnostics)
	return resp.State
}

func assertLiveAttrs(ctx context.Context, t *testing.T, state tfsdk.State, checks []LiveAttrCheck) {
	t.Helper()
	for _, check := range checks {
		var got types.String
		if diags := state.GetAttribute(ctx, check.Path, &got); diags.HasError() {
			t.Fatalf("get %s: %v", check.Path, diags)
		}
		switch {
		case check.Want != "":
			if got.ValueString() != check.Want {
				t.Errorf("%s = %q, want %q", check.Path, got.ValueString(), check.Want)
			}
		case got.IsNull() || got.ValueString() == "":
			t.Errorf("%s not populated by Read", check.Path)
		}
	}
}
