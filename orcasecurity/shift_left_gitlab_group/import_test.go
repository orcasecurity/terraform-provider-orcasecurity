package shift_left_gitlab_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func runImport(t *testing.T, id string) (resourceModel, bool) {
	t.Helper()
	ctx := context.Background()
	r := &gitlabGroupResource{}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sch := schemaResp.Schema

	resp := resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: sch,
			Raw:    tftypes.NewValue(sch.Type().TerraformType(ctx), nil),
		},
	}
	r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)

	var m resourceModel
	resp.State.Get(ctx, &m)
	return m, resp.Diagnostics.HasError()
}

func TestImportState_UUIDGoesToID(t *testing.T) {
	m, hasErr := runImport(t, "inst-1/019f90dd-0f28-7f15-b106-387f0ba7cee8")
	if hasErr {
		t.Fatal("unexpected error diagnostics")
	}
	if got := m.ID.ValueString(); got != "019f90dd-0f28-7f15-b106-387f0ba7cee8" {
		t.Errorf("id = %q, want the uuid", got)
	}
	if !m.GitlabGroupID.IsNull() {
		t.Errorf("gitlab_group_id = %v, want null", m.GitlabGroupID)
	}
}

func TestImportState_NumericGoesToGroupID(t *testing.T) {
	m, hasErr := runImport(t, "inst-1/82593918")
	if hasErr {
		t.Fatal("unexpected error diagnostics")
	}
	if got := m.InstallationID.ValueString(); got != "inst-1" {
		t.Errorf("installation_id = %q, want inst-1", got)
	}
	if got := m.GitlabGroupID.ValueInt64(); got != 82593918 {
		t.Errorf("gitlab_group_id = %d, want 82593918", got)
	}
	if !m.ID.IsNull() {
		t.Errorf("id = %v, want null", m.ID)
	}
}

func TestImportState_NonNumericNonUUIDErrors(t *testing.T) {
	if _, hasErr := runImport(t, "inst-1/not-a-uuid-or-number"); !hasErr {
		t.Error("expected error diagnostics for non-numeric non-uuid right-hand side")
	}
}

func TestImportState_MalformedIDErrors(t *testing.T) {
	for _, id := range []string{"noslash", "/only-right", "only-left/", ""} {
		if _, hasErr := runImport(t, id); !hasErr {
			t.Errorf("id %q: expected error diagnostics, got none", id)
		}
	}
}
