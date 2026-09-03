package shift_left_unit

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func runAzureImport(t *testing.T, id string) (azureAccountModel, bool) {
	t.Helper()
	ctx := context.Background()
	r := NewAzureDevopsAccountResource().(resource.ResourceWithImportState)

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

	var m azureAccountModel
	resp.State.Get(ctx, &m)
	return m, resp.Diagnostics.HasError()
}

func TestAzureImportState_UUIDGoesToID(t *testing.T) {
	m, hasErr := runAzureImport(t, "inst-1/019f90dd-0f28-7f15-b106-387f0ba7cee8")
	if hasErr {
		t.Fatal("unexpected error diagnostics")
	}
	if got := m.InstallationID.ValueString(); got != "inst-1" {
		t.Errorf("installation_id = %q, want inst-1", got)
	}
	if got := m.ID.ValueString(); got != "019f90dd-0f28-7f15-b106-387f0ba7cee8" {
		t.Errorf("id = %q, want the uuid", got)
	}
	if !m.AccountName.IsNull() {
		t.Errorf("account_name = %v, want null", m.AccountName)
	}
}

func TestAzureImportState_NameGoesToNameAttr(t *testing.T) {
	m, hasErr := runAzureImport(t, "inst-1/rahg0-azdevops")
	if hasErr {
		t.Fatal("unexpected error diagnostics")
	}
	if got := m.InstallationID.ValueString(); got != "inst-1" {
		t.Errorf("installation_id = %q, want inst-1", got)
	}
	if got := m.AccountName.ValueString(); got != "rahg0-azdevops" {
		t.Errorf("account_name = %q, want rahg0-azdevops", got)
	}
	if !m.ID.IsNull() {
		t.Errorf("id = %v, want null", m.ID)
	}
}

func TestAzureImportState_MalformedIDErrors(t *testing.T) {
	for _, id := range []string{"noslash", "/only-right", "only-left/", ""} {
		if _, hasErr := runAzureImport(t, id); !hasErr {
			t.Errorf("id %q: expected error diagnostics, got none", id)
		}
	}
}
