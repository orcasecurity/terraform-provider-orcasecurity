package shift_left_integration

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// scmUnitType is a trimmed unit object: two writables, two volatiles, plus a
// nested settings object. The volatile names missing from this type
// (integration_status, scm_posture_policy_id) exercise the skip on absent keys.
func scmUnitType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"project_id":                    tftypes.String,
			"installation_mode":             tftypes.String,
			"configuration_settings":        settingsType(),
			"integrated_repositories_count": tftypes.Number,
			"scan_all_state":                tftypes.String,
		},
	}
}

func settingsType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{"pr_summary_appendix": tftypes.String}}
}

func settingsValue(appendix tftypes.Value) tftypes.Value {
	return tftypes.NewValue(settingsType(), map[string]tftypes.Value{"pr_summary_appendix": appendix})
}

func unitValue(project, mode, appendix tftypes.Value, count, scanState tftypes.Value) tftypes.Value {
	return tftypes.NewValue(scmUnitType(), map[string]tftypes.Value{
		"project_id":                    project,
		"installation_mode":             mode,
		"configuration_settings":        settingsValue(appendix),
		"integrated_repositories_count": count,
		"scan_all_state":                scanState,
	})
}

func str(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
func num(n int64) tftypes.Value  { return tftypes.NewValue(tftypes.Number, n) }
func unknownStr() tftypes.Value  { return tftypes.NewValue(tftypes.String, tftypes.UnknownValue) }
func unknownNum() tftypes.Value  { return tftypes.NewValue(tftypes.Number, tftypes.UnknownValue) }
func settledState() tftypes.Value {
	return unitValue(str("proj-a"), str("SELECTED_REPOSITORIES"), str("hi"), num(3), str("DONE"))
}
func hydratedNoopPlan() tftypes.Value {
	// What attribute modifiers produce for an omitted-everything config on
	// TF 1.0–1.3: writables carried from state, volatiles still unknown.
	return unitValue(str("proj-a"), str("SELECTED_REPOSITORIES"), str("hi"), unknownNum(), unknownStr())
}

func settle(t *testing.T, stateRaw, planRaw tftypes.Value) map[string]tftypes.Value {
	t.Helper()
	req := resource.ModifyPlanRequest{
		State: tfsdk.State{Raw: stateRaw},
		Plan:  tfsdk.Plan{Raw: planRaw},
	}
	resp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Raw: planRaw}}
	settleVolatileAttrs(req, resp)
	var out map[string]tftypes.Value
	if err := resp.Plan.Raw.As(&out); err != nil {
		t.Fatalf("settled plan: %v", err)
	}
	return out
}

func TestSettleVolatiles_CarriesOnNoopPlan(t *testing.T) {
	out := settle(t, settledState(), hydratedNoopPlan())
	if !out["integrated_repositories_count"].Equal(num(3)) || !out["scan_all_state"].Equal(str("DONE")) {
		t.Fatalf("no-op plan must carry volatiles from state, got %v / %v",
			out["integrated_repositories_count"], out["scan_all_state"])
	}
}

func TestSettleVolatiles_KeepsKnownCarriedValuesOnNoopPlan(t *testing.T) {
	// TF >= 1.4 arrives with volatiles already carried by core; they must stay put.
	out := settle(t, settledState(), settledState())
	if !out["integrated_repositories_count"].Equal(num(3)) || !out["scan_all_state"].Equal(str("DONE")) {
		t.Fatalf("carried volatiles must survive a no-op plan, got %v / %v",
			out["integrated_repositories_count"], out["scan_all_state"])
	}
}

func TestSettleVolatiles_UnknownOnWritableChange(t *testing.T) {
	plan := unitValue(str("proj-a"), str("SCAN_ALL_INCLUDE_FUTURE"), str("hi"), num(3), str("DONE"))
	out := settle(t, settledState(), plan)
	if out["integrated_repositories_count"].IsKnown() || out["scan_all_state"].IsKnown() {
		t.Fatalf("writable change must leave volatiles unknown, got %v / %v",
			out["integrated_repositories_count"], out["scan_all_state"])
	}
}

func TestSettleVolatiles_UnknownOnUnknownWritable(t *testing.T) {
	// Cross-resource reference: project_id unresolved at plan time.
	plan := unitValue(unknownStr(), str("SELECTED_REPOSITORIES"), str("hi"), unknownNum(), unknownStr())
	out := settle(t, settledState(), plan)
	if out["integrated_repositories_count"].IsKnown() || out["scan_all_state"].IsKnown() {
		t.Fatal("unknown writable must leave volatiles unknown")
	}
}

func TestSettleVolatiles_UnknownOnNestedChange(t *testing.T) {
	// A change buried in the settings object is still a write.
	plan := unitValue(str("proj-a"), str("SELECTED_REPOSITORIES"), str("changed"), num(3), str("DONE"))
	out := settle(t, settledState(), plan)
	if out["integrated_repositories_count"].IsKnown() || out["scan_all_state"].IsKnown() {
		t.Fatal("nested writable change must leave volatiles unknown")
	}
}

// A pending legacy-mode migration reaches this code as a plan already rewritten
// by installationModePlanModifier (state SCAN_ALL, plan SELECTED_REPOSITORIES),
// so the migration write is detected as a plain mode diff — even though the
// framework left the volatile values known because the proposed plan equalled
// prior state before attribute modifiers ran.
func TestSettleVolatiles_UnknownOnLegacyModeMigration(t *testing.T) {
	state := unitValue(str("proj-a"), str("SCAN_ALL"), str("hi"), num(3), str("RUNNING"))
	plan := unitValue(str("proj-a"), str("SELECTED_REPOSITORIES"), str("hi"), num(3), str("RUNNING"))
	out := settle(t, state, plan)
	if out["integrated_repositories_count"].IsKnown() || out["scan_all_state"].IsKnown() {
		t.Fatal("pending SCAN_ALL migration must leave volatiles unknown")
	}
}

func TestSettleVolatiles_NoopOnCreate(t *testing.T) {
	plan := unitValue(str("proj-a"), str("SELECTED_REPOSITORIES"), str("hi"), unknownNum(), unknownStr())
	out := settle(t, tftypes.NewValue(scmUnitType(), nil), plan)
	if out["integrated_repositories_count"].IsKnown() || out["scan_all_state"].IsKnown() {
		t.Fatal("create must leave volatiles unknown (no state to carry)")
	}
}
