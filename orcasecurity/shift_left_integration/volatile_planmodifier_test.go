package shift_left_integration

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func scmUnitType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"project_id":                    tftypes.String,
			"installation_mode":             tftypes.String,
			"integrated_repositories_count": tftypes.Number,
			"scan_all_state":                tftypes.String,
		},
	}
}

func TestWritableConfigMatchesState_SkipsNullComputed(t *testing.T) {
	objType := scmUnitType()
	state := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, 3),
		"scan_all_state":                tftypes.NewValue(tftypes.String, "DONE"),
	})
	// Config omits computed volatiles (null) and restates the known writables.
	config := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, nil),
		"scan_all_state":                tftypes.NewValue(tftypes.String, nil),
	})
	if !writableConfigMatchesState(config, state) {
		t.Fatal("expected match when only computed config leaves are null")
	}
}

func TestWritableConfigMatchesState_UnknownWritable(t *testing.T) {
	objType := scmUnitType()
	state := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, 3),
		"scan_all_state":                tftypes.NewValue(tftypes.String, "DONE"),
	})
	// Cross-resource reference: project_id unknown at plan time.
	config := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, nil),
		"scan_all_state":                tftypes.NewValue(tftypes.String, nil),
	})
	if writableConfigMatchesState(config, state) {
		t.Fatal("unknown writable must not match — leave volatile unknown")
	}
}

func TestWritableConfigMatchesState_DetectsWritableChange(t *testing.T) {
	objType := scmUnitType()
	state := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, 3),
		"scan_all_state":                tftypes.NewValue(tftypes.String, "DONE"),
	})
	config := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SCAN_ALL_INCLUDE_FUTURE"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, nil),
		"scan_all_state":                tftypes.NewValue(tftypes.String, nil),
	})
	if writableConfigMatchesState(config, state) {
		t.Fatal("expected mismatch when a known configured value changed")
	}
}

func TestCarryVolatileForward(t *testing.T) {
	objType := scmUnitType()
	state := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, 3),
		"scan_all_state":                tftypes.NewValue(tftypes.String, "DONE"),
	})
	noopConfig := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, nil),
		"scan_all_state":                tftypes.NewValue(tftypes.String, nil),
	})
	if !carryVolatileForward(state, noopConfig, true) {
		t.Fatal("expected carry-forward on no-op config")
	}
	if carryVolatileForward(state, noopConfig, false) {
		t.Fatal("must not carry when plan value is already known")
	}
	if carryVolatileForward(tftypes.NewValue(objType, nil), noopConfig, true) {
		t.Fatal("must not carry on create (null state)")
	}

	unknownProject := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, nil),
		"scan_all_state":                tftypes.NewValue(tftypes.String, nil),
	})
	if carryVolatileForward(state, unknownProject, true) {
		t.Fatal("must not carry when a writable config value is unknown")
	}

	changed := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SCAN_ALL_INCLUDE_FUTURE"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, nil),
		"scan_all_state":                tftypes.NewValue(tftypes.String, nil),
	})
	if carryVolatileForward(state, changed, true) {
		t.Fatal("must not carry when a known configured value changed")
	}
}

func TestVolatileInt64_CarryOnNoopConfig(t *testing.T) {
	objType := scmUnitType()
	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, 3),
		"scan_all_state":                tftypes.NewValue(tftypes.String, "DONE"),
	})
	configRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, nil),
		"scan_all_state":                tftypes.NewValue(tftypes.String, nil),
	})

	req := planmodifier.Int64Request{
		State:      tfsdk.State{Raw: stateRaw},
		Config:     tfsdk.Config{Raw: configRaw},
		StateValue: types.Int64Value(3),
		PlanValue:  types.Int64Unknown(),
	}
	resp := planmodifier.Int64Response{PlanValue: req.PlanValue}
	VolatileInt64().PlanModifyInt64(context.Background(), req, &resp)
	if resp.PlanValue.IsUnknown() || resp.PlanValue.ValueInt64() != 3 {
		t.Fatalf("noop config should carry state count, got %#v", resp.PlanValue)
	}
}

func TestVolatileInt64_LeaveUnknownOnUnknownWritable(t *testing.T) {
	objType := scmUnitType()
	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, 3),
		"scan_all_state":                tftypes.NewValue(tftypes.String, "DONE"),
	})
	configRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, nil),
		"scan_all_state":                tftypes.NewValue(tftypes.String, nil),
	})

	req := planmodifier.Int64Request{
		State:      tfsdk.State{Raw: stateRaw},
		Config:     tfsdk.Config{Raw: configRaw},
		StateValue: types.Int64Value(3),
		PlanValue:  types.Int64Unknown(),
	}
	resp := planmodifier.Int64Response{PlanValue: req.PlanValue}
	VolatileInt64().PlanModifyInt64(context.Background(), req, &resp)
	if !resp.PlanValue.IsUnknown() {
		t.Fatalf("unknown writable must leave volatile unknown, got %#v", resp.PlanValue)
	}
}

func TestLegacyModeMigrationPending(t *testing.T) {
	objType := scmUnitType()
	build := func(mode tftypes.Value) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
			"installation_mode":             mode,
			"integrated_repositories_count": tftypes.NewValue(tftypes.Number, 3),
			"scan_all_state":                tftypes.NewValue(tftypes.String, "DONE"),
		})
	}
	if !legacyModeMigrationPending(build(tftypes.NewValue(tftypes.String, "SCAN_ALL"))) {
		t.Fatal("legacy SCAN_ALL in state means the next write migrates it")
	}
	if legacyModeMigrationPending(build(tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"))) {
		t.Fatal("a settled mode is not a pending migration")
	}
	if legacyModeMigrationPending(build(tftypes.NewValue(tftypes.String, "SCAN_ALL_INCLUDE_FUTURE"))) {
		t.Fatal("SCAN_ALL_INCLUDE_FUTURE is not remapped by the write path")
	}
	if legacyModeMigrationPending(build(tftypes.NewValue(tftypes.String, nil))) {
		t.Fatal("a null mode carries no migration")
	}
	if legacyModeMigrationPending(tftypes.NewValue(objType, nil)) {
		t.Fatal("create (null state) carries no migration")
	}
}

// A pending legacy-mode migration must force the volatile attribute unknown even
// though its planned value is known (the framework only marks Computed attributes
// unknown when the proposed plan already differs from prior state, which the
// modifier-generated migration write does not).
func TestVolatileInt64_ForceUnknownOnLegacyModeMigration(t *testing.T) {
	objType := scmUnitType()
	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SCAN_ALL"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, 3),
		"scan_all_state":                tftypes.NewValue(tftypes.String, "RUNNING"),
	})
	configRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, nil),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, nil),
		"scan_all_state":                tftypes.NewValue(tftypes.String, nil),
	})

	req := planmodifier.Int64Request{
		State:      tfsdk.State{Raw: stateRaw},
		Config:     tfsdk.Config{Raw: configRaw},
		StateValue: types.Int64Value(3),
		// Proposed plan equals prior state, so the framework left the value known.
		PlanValue: types.Int64Value(3),
	}
	resp := planmodifier.Int64Response{PlanValue: req.PlanValue}
	VolatileInt64().PlanModifyInt64(context.Background(), req, &resp)
	if !resp.PlanValue.IsUnknown() {
		t.Fatalf("pending SCAN_ALL migration must leave volatile unknown, got %#v", resp.PlanValue)
	}
}

func TestVolatileInt64_LeaveUnknownOnWritableChange(t *testing.T) {
	objType := scmUnitType()
	stateRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SELECTED_REPOSITORIES"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, 3),
		"scan_all_state":                tftypes.NewValue(tftypes.String, "DONE"),
	})
	configRaw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"project_id":                    tftypes.NewValue(tftypes.String, "proj-a"),
		"installation_mode":             tftypes.NewValue(tftypes.String, "SCAN_ALL_INCLUDE_FUTURE"),
		"integrated_repositories_count": tftypes.NewValue(tftypes.Number, nil),
		"scan_all_state":                tftypes.NewValue(tftypes.String, nil),
	})

	req := planmodifier.Int64Request{
		State:      tfsdk.State{Raw: stateRaw},
		Config:     tfsdk.Config{Raw: configRaw},
		StateValue: types.Int64Value(3),
		PlanValue:  types.Int64Unknown(),
	}
	resp := planmodifier.Int64Response{PlanValue: req.PlanValue}
	VolatileInt64().PlanModifyInt64(context.Background(), req, &resp)
	if !resp.PlanValue.IsUnknown() {
		t.Fatalf("writable change must leave volatile unknown, got %#v", resp.PlanValue)
	}
}
