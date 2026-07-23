package shift_left_policy

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func baseBuiltin() *shiftLeftPolicyResourceModel {
	return &shiftLeftPolicyResourceModel{
		ID:                       types.StringValue("p1"),
		Type:                     types.StringValue("licenses"),
		Name:                     types.StringValue("OSS Licenses Policy"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		Builtin:                  types.BoolValue(true),
	}
}

func TestBuiltinGuard_ProjectsOnlyChangeAllowed(t *testing.T) {
	state := baseBuiltin()
	plan := baseBuiltin()
	plan.ProjectsIds = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("proj-1"), types.StringValue("proj-2")})

	if field, changed := builtinLockedFieldChanged(plan, state); changed {
		t.Fatalf("expected projects-only change to be allowed, but field %q flagged", field)
	}
}

func TestBuiltinGuard_NameChangeRejected(t *testing.T) {
	state := baseBuiltin()
	plan := baseBuiltin()
	plan.Name = types.StringValue("renamed")

	field, changed := builtinLockedFieldChanged(plan, state)
	if !changed || field != "name" {
		t.Fatalf("expected name change to be rejected, got field=%q changed=%v", field, changed)
	}
}

// Non-name built-in fields remain updatable through the guard.
func TestBuiltinGuard_ApiUpdatableFieldsAllowed(t *testing.T) {
	state := baseBuiltin()
	plan := baseBuiltin()
	plan.Description = types.StringValue("new description")
	plan.Disabled = types.BoolValue(true)
	plan.WarnMode = types.BoolValue(true)
	plan.PriorityFailureThreshold = types.StringValue("MEDIUM")
	plan.Licenses = &licensesBlockModel{AllControls: types.BoolValue(true)}
	state.Licenses = &licensesBlockModel{AllControls: types.BoolValue(false)}

	if field, changed := builtinLockedFieldChanged(plan, state); changed {
		t.Fatalf("expected API-updatable fields to be allowed on builtins, but field %q flagged", field)
	}
}

func TestBuiltinGuard_ContainerFeatureScopeRejected(t *testing.T) {
	state := baseBuiltin()
	state.Type = types.StringValue("container_image")
	state.ContainerImage = &containerImageBlockModel{
		FeatureScope: []types.String{types.StringValue("vulnerabilities")},
	}
	plan := baseBuiltin()
	plan.Type = types.StringValue("container_image")
	plan.ContainerImage = &containerImageBlockModel{
		FeatureScope: []types.String{types.StringValue("vulnerabilities"), types.StringValue("custom")},
	}

	field, changed := builtinLockedFieldChanged(plan, state)
	if !changed || field != "container_image.feature_scope" {
		t.Fatalf("expected feature_scope change to be rejected, got field=%q changed=%v", field, changed)
	}
}

func TestBuiltinGuard_ScmPostureDescriptionAndScopeRejected(t *testing.T) {
	state := baseBuiltin()
	state.Type = types.StringValue("scm_posture")
	plan := baseBuiltin()
	plan.Type = types.StringValue("scm_posture")
	plan.Description = types.StringValue("changed")

	field, changed := builtinLockedFieldChanged(plan, state)
	if !changed || field != "description" {
		t.Fatalf("expected scm_posture description change to be rejected, got field=%q changed=%v", field, changed)
	}

	plan = baseBuiltin()
	plan.Type = types.StringValue("scm_posture")
	plan.ScmPosture = &scmPostureBlockModel{
		Scope: []scmScopeEntryModel{{Key: types.StringValue("gitlab_groups"), Ids: []types.String{types.StringValue("g1")}}},
	}
	state = baseBuiltin()
	state.Type = types.StringValue("scm_posture")
	state.ScmPosture = &scmPostureBlockModel{}

	field, changed = builtinLockedFieldChanged(plan, state)
	if !changed || field != "scm_posture.scope" {
		t.Fatalf("expected scm_posture scope change to be rejected, got field=%q changed=%v", field, changed)
	}
}

func TestBuiltinGuard_ScmPostureControlsAllowed(t *testing.T) {
	state := baseBuiltin()
	state.Type = types.StringValue("scm_posture")
	state.ScmPosture = &scmPostureBlockModel{
		Controls: []scmControlModel{{ID: types.StringValue("c1"), Priority: types.StringValue("HIGH"), Disabled: types.BoolValue(false)}},
	}
	plan := baseBuiltin()
	plan.Type = types.StringValue("scm_posture")
	plan.ScmPosture = &scmPostureBlockModel{
		Controls: []scmControlModel{{ID: types.StringValue("c1"), Priority: types.StringValue("LOW"), Disabled: types.BoolValue(true)}},
	}

	if field, changed := builtinLockedFieldChanged(plan, state); changed {
		t.Fatalf("expected scm_posture control override to be allowed, but field %q flagged", field)
	}
}
