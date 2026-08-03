package shift_left_integration

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateScmBindingPlan_C1(t *testing.T) {
	var diags diag.Diagnostics
	ValidateScmBindingPlan(&ScmConfigFields{
		DefaultPolicies: types.BoolValue(true),
		PoliciesIds:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-1")}),
		ProjectID:       types.StringNull(),
	}, &diags)
	if !diags.HasError() {
		t.Fatal("expected error for default_policies=true + policies_ids")
	}
}

func TestValidateScmBindingPlan_C2(t *testing.T) {
	var diags diag.Diagnostics
	ValidateScmBindingPlan(&ScmConfigFields{
		DefaultPolicies: types.BoolValue(false),
		PoliciesIds:     types.SetNull(types.StringType),
		ProjectID:       types.StringNull(),
	}, &diags)
	if !diags.HasError() {
		t.Fatal("expected error for default_policies=false alone")
	}
}

func TestValidateScmBindingPlan_FalseWithPoliciesOK(t *testing.T) {
	var diags diag.Diagnostics
	ValidateScmBindingPlan(&ScmConfigFields{
		DefaultPolicies: types.BoolValue(false),
		PoliciesIds:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-1")}),
		ProjectID:       types.StringNull(),
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected: %v", diags)
	}
}

func TestValidateScmBindingPlan_TrueAloneOK(t *testing.T) {
	var diags diag.Diagnostics
	ValidateScmBindingPlan(&ScmConfigFields{
		DefaultPolicies: types.BoolValue(true),
		PoliciesIds:     types.SetNull(types.StringType),
		ProjectID:       types.StringNull(),
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected: %v", diags)
	}
}

func TestValidateScmBindingPlan_FalseWithProjectOK(t *testing.T) {
	var diags diag.Diagnostics
	ValidateScmBindingPlan(&ScmConfigFields{
		DefaultPolicies: types.BoolValue(false),
		PoliciesIds:     types.SetNull(types.StringType),
		ProjectID:       types.StringValue("proj-1"),
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected: %v", diags)
	}
}

// Every attribute settleVolatileAttrs manages must exist in the shared unit
// schema as computed-only; a rename on either side would silently orphan the
// attribute from the resource-level settling.
func TestSharedScmConfigAttributes_VolatileStatusFields(t *testing.T) {
	attrs := SharedScmConfigAttributes("name")
	for _, name := range ScmVolatileAttrNames() {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if !attr.IsComputed() || attr.IsOptional() || attr.IsRequired() {
			t.Errorf("%s must be computed-only (server-owned)", name)
		}
	}
}
