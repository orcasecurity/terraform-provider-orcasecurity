package shift_left_integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

func TestSharedScmConfigAttributes_VolatileStatusFields(t *testing.T) {
	attrs := SharedScmConfigAttributes("name")
	for _, name := range []string{"integration_status", "scan_all_state", "integrated_repositories_count", "scm_posture_policy_id"} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if !attrHasVolatileModifier(attr) {
			t.Errorf("%s must use a volatile plan modifier", name)
		}
	}
}

func attrHasVolatileModifier(attr rschema.Attribute) bool {
	var modifiers []any
	switch a := attr.(type) {
	case rschema.StringAttribute:
		for _, m := range a.PlanModifiers {
			modifiers = append(modifiers, m)
		}
	case rschema.Int64Attribute:
		for _, m := range a.PlanModifiers {
			modifiers = append(modifiers, m)
		}
	default:
		return false
	}
	for _, m := range modifiers {
		if strings.Contains(fmt.Sprintf("%T", m), "volatile") {
			return true
		}
	}
	return false
}
