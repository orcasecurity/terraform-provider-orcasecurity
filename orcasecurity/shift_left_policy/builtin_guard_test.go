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

func TestBuiltinGuard_DescriptionChangeRejected(t *testing.T) {
	state := baseBuiltin()
	plan := baseBuiltin()
	plan.Description = types.StringValue("new description")

	field, changed := builtinNonProjectFieldChanged(plan, state)
	if !changed || field != "description" {
		t.Fatalf("expected description change to be rejected, got field=%q changed=%v", field, changed)
	}
}

func TestBuiltinGuard_DisabledChangeRejected(t *testing.T) {
	state := baseBuiltin()
	plan := baseBuiltin()
	plan.Disabled = types.BoolValue(true)

	field, changed := builtinNonProjectFieldChanged(plan, state)
	if !changed || field != "disabled" {
		t.Fatalf("expected disabled change to be rejected, got field=%q changed=%v", field, changed)
	}
}

func TestBuiltinGuard_WarnModeChangeRejected(t *testing.T) {
	state := baseBuiltin()
	plan := baseBuiltin()
	plan.WarnMode = types.BoolValue(true)

	field, changed := builtinNonProjectFieldChanged(plan, state)
	if !changed || field != "warn_mode" {
		t.Fatalf("expected warn_mode change to be rejected, got field=%q changed=%v", field, changed)
	}
}

func TestBuiltinGuard_PriorityFailureThresholdChangeRejected(t *testing.T) {
	state := baseBuiltin()
	plan := baseBuiltin()
	plan.PriorityFailureThreshold = types.StringValue("MEDIUM")

	field, changed := builtinNonProjectFieldChanged(plan, state)
	if !changed || field != "priority_failure_threshold" {
		t.Fatalf("expected priority_failure_threshold change to be rejected, got field=%q changed=%v", field, changed)
	}
}

func TestBuiltinGuard_IacControlChangeRejected(t *testing.T) {
	state := baseBuiltin()
	state.Iac = &iacBlockModel{
		AllControls: types.BoolValue(false),
		Controls: []iacControlModel{
			{
				baseControlModel: baseControlModel{
					ID:       types.StringValue("ctrl-1"),
					Disabled: types.BoolValue(false),
				},
			},
		},
	}
	plan := baseBuiltin()
	plan.Iac = &iacBlockModel{
		AllControls: types.BoolValue(false),
		Controls: []iacControlModel{
			{
				baseControlModel: baseControlModel{
					ID:       types.StringValue("ctrl-1"),
					Disabled: types.BoolValue(true),
				},
			},
		},
	}

	field, changed := builtinNonProjectFieldChanged(plan, state)
	if !changed || field != "iac" {
		t.Fatalf("expected iac control change to be rejected, got field=%q changed=%v", field, changed)
	}
}

func TestBuiltinGuard_SastControlChangeRejected(t *testing.T) {
	state := baseBuiltin()
	state.Sast = &sastBlockModel{AllControls: types.BoolValue(true)}
	plan := baseBuiltin()
	plan.Sast = &sastBlockModel{AllControls: types.BoolValue(false)}

	field, changed := builtinNonProjectFieldChanged(plan, state)
	if !changed || field != "sast" {
		t.Fatalf("expected sast control change to be rejected, got field=%q changed=%v", field, changed)
	}
}

func TestBuiltinGuard_LicensesControlChangeRejected(t *testing.T) {
	state := baseBuiltin()
	state.Licenses = &licensesBlockModel{AllControls: types.BoolValue(true)}
	plan := baseBuiltin()
	plan.Licenses = &licensesBlockModel{AllControls: types.BoolValue(false)}

	field, changed := builtinNonProjectFieldChanged(plan, state)
	if !changed || field != "licenses" {
		t.Fatalf("expected licenses control change to be rejected, got field=%q changed=%v", field, changed)
	}
}
