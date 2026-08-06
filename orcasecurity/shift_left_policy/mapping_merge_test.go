package shift_left_policy

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMergeStateFromPlan_AllControlsClearsControls(t *testing.T) {
	state := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac: &iacBlockModel{
			Controls: []iacControlModel{
				{baseControlModel: baseControlModel{ID: types.StringValue("from-api")}},
			},
		},
	}
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac:  &iacBlockModel{AllControls: types.BoolValue(true)},
	}

	mergeStateFromPlan(state, plan)
	if !state.Iac.AllControls.ValueBool() {
		t.Error("expected all_controls to be carried from plan")
	}
	if len(state.Iac.Controls) != 0 {
		t.Errorf("expected controls cleared when all_controls is set, got %+v", state.Iac.Controls)
	}
}

func TestMergeStateFromPlan_TitleReferenceKeepsIDNull(t *testing.T) {
	state := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac: &iacBlockModel{
			Controls: []iacControlModel{
				{baseControlModel: baseControlModel{
					ID:       types.StringValue("api-resolved-id"),
					Priority: types.StringValue("HIGH"),
					Disabled: types.BoolValue(true),
				}},
			},
		},
	}
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac: &iacBlockModel{
			Controls: []iacControlModel{
				{baseControlModel: baseControlModel{
					Title:    types.StringValue("Some control title"),
					Priority: types.StringValue("HIGH"),
					Disabled: types.BoolValue(true),
				}},
			},
		},
	}

	mergeStateFromPlan(state, plan)
	if !state.Iac.Controls[0].ID.IsNull() {
		t.Errorf("expected id to stay null when plan referenced control by title, got %s", state.Iac.Controls[0].ID.ValueString())
	}
}

// The API decides the order it returns controls in. Overlaying prior state by position would move
// one control's overrides onto another, silently rewriting the policy the operator configured.

func TestMergeStateFromPlan_PairsControlsByIDNotPosition(t *testing.T) {
	iacControl := func(id, priority string, disabled bool) iacControlModel {
		return iacControlModel{baseControlModel: baseControlModel{
			ID:       types.StringValue(id),
			Priority: types.StringValue(priority),
			Disabled: types.BoolValue(disabled),
		}}
	}
	state := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac: &iacBlockModel{Controls: []iacControlModel{
			iacControl("control-b", "LOW", false),
			iacControl("control-a", "LOW", false),
		}},
	}
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("iac"),
		Iac: &iacBlockModel{Controls: []iacControlModel{
			iacControl("control-a", "CRITICAL", true),
			iacControl("control-b", "MEDIUM", false),
		}},
	}

	mergeStateFromPlan(state, plan)

	got := map[string]iacControlModel{}
	for _, control := range state.Iac.Controls {
		got[control.ID.ValueString()] = control
	}
	if p := got["control-a"].Priority.ValueString(); p != "CRITICAL" {
		t.Errorf("control-a priority = %q, want CRITICAL from its own configured override", p)
	}
	if !got["control-a"].Disabled.ValueBool() {
		t.Error("control-a must keep its own disabled override")
	}
	if p := got["control-b"].Priority.ValueString(); p != "MEDIUM" {
		t.Errorf("control-b priority = %q, want MEDIUM from its own configured override", p)
	}
	if got["control-b"].Disabled.ValueBool() {
		t.Error("control-b must not inherit control-a's disabled override")
	}
}

// A control the API no longer returns must not shift the remaining controls' overrides along.

func TestMergeStateFromPlan_DroppedApiControlDoesNotShiftOverrides(t *testing.T) {
	state := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("scm_posture"),
		ScmPosture: &scmPostureBlockModel{Controls: []scmControlModel{
			{ID: types.StringValue("kept"), Priority: types.StringValue("LOW")},
		}},
	}
	plan := &shiftLeftPolicyResourceModel{
		Type: types.StringValue("scm_posture"),
		ScmPosture: &scmPostureBlockModel{Controls: []scmControlModel{
			{ID: types.StringValue("removed-server-side"), Priority: types.StringValue("CRITICAL")},
			{ID: types.StringValue("kept"), Priority: types.StringValue("HIGH")},
		}},
	}

	mergeStateFromPlan(state, plan)
	if p := state.ScmPosture.Controls[0].Priority.ValueString(); p != "HIGH" {
		t.Errorf("kept control priority = %q, want HIGH (its own override, not the removed control's)", p)
	}
}
