package shift_left_policy

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func baseBuiltin() *shiftLeftPolicyResourceModel {
	return &shiftLeftPolicyResourceModel{
		ID:                       types.StringValue("p1"),
		Type:                     types.StringValue("sca"),
		Name:                     types.StringValue("Malicious Packages"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		Builtin:                  types.BoolValue(true),
	}
}

func TestBuiltinGuard_ProjectsOnlyChangeAllowed(t *testing.T) {
	state := baseBuiltin()
	plan := baseBuiltin()
	plan.ProjectsIds = []types.String{types.StringValue("proj-1"), types.StringValue("proj-2")}

	field, changed := builtinNonProjectFieldChanged(plan, state)
	if changed {
		t.Fatalf("expected projects-only change to be allowed, but field %q flagged", field)
	}
}

func TestBuiltinGuard_NameChangeRejected(t *testing.T) {
	state := baseBuiltin()
	plan := baseBuiltin()
	plan.Name = types.StringValue("renamed")

	field, changed := builtinNonProjectFieldChanged(plan, state)
	if !changed || field != "name" {
		t.Fatalf("expected name change to be rejected, got field=%q changed=%v", field, changed)
	}
}
