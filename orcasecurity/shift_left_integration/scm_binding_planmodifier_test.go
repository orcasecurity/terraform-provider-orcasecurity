package shift_left_integration

import (
	"context"
	"testing"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func bindingSchema() rschema.Schema {
	return rschema.Schema{
		Attributes: map[string]rschema.Attribute{
			"policies_ids":     rschema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true},
			"default_policies": rschema.BoolAttribute{Optional: true, Computed: true},
			"project_id":       rschema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func bindingObjectType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"policies_ids":     tftypes.Set{ElementType: tftypes.String},
			"default_policies": tftypes.Bool,
			"project_id":       tftypes.String,
		},
	}
}

type bindingValues struct {
	policies    []string // nil means null
	defaultPols *bool    // nil means null
	projectID   *string  // nil means null
}

func bindingRaw(v bindingValues) tftypes.Value {
	setType := tftypes.Set{ElementType: tftypes.String}
	policies := tftypes.NewValue(setType, nil)
	if v.policies != nil {
		elems := make([]tftypes.Value, len(v.policies))
		for i, p := range v.policies {
			elems[i] = tftypes.NewValue(tftypes.String, p)
		}
		policies = tftypes.NewValue(setType, elems)
	}
	defaultPols := tftypes.NewValue(tftypes.Bool, nil)
	if v.defaultPols != nil {
		defaultPols = tftypes.NewValue(tftypes.Bool, *v.defaultPols)
	}
	projectID := tftypes.NewValue(tftypes.String, nil)
	if v.projectID != nil {
		projectID = tftypes.NewValue(tftypes.String, *v.projectID)
	}
	return tftypes.NewValue(bindingObjectType(), map[string]tftypes.Value{
		"policies_ids":     policies,
		"default_policies": defaultPols,
		"project_id":       projectID,
	})
}

func boolPtr(b bool) *bool          { return &b }
func strPtr(s string) *string       { return &s }
func nullBindingRaw() tftypes.Value { return tftypes.NewValue(bindingObjectType(), nil) }

func bindingConfig(v bindingValues) tfsdk.Config {
	return tfsdk.Config{Schema: bindingSchema(), Raw: bindingRaw(v)}
}

func bindingState(raw tftypes.Value) tfsdk.State {
	return tfsdk.State{Schema: bindingSchema(), Raw: raw}
}

func attachedPolicies(ids ...string) types.Set {
	elems := make([]types.String, len(ids))
	for i, id := range ids {
		elems[i] = types.StringValue(id)
	}
	set, _ := types.SetValueFrom(context.Background(), types.StringType, elems)
	return set
}

// --- policies_ids ---

func runPoliciesModifier(t *testing.T, config tfsdk.Config, stateRaw tftypes.Value, stateValue types.Set) types.Set {
	t.Helper()
	req := planmodifier.SetRequest{
		Config:      config,
		ConfigValue: types.SetNull(types.StringType),
		State:       bindingState(stateRaw),
		StateValue:  stateValue,
		PlanValue:   types.SetUnknown(types.StringType),
	}
	resp := planmodifier.SetResponse{PlanValue: req.PlanValue}
	PoliciesIDsPlanModifier().PlanModifySet(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	return resp.PlanValue
}

func TestPoliciesIDsModifier_OmittedCarriesState(t *testing.T) {
	state := bindingValues{policies: []string{"pol-1"}}
	got := runPoliciesModifier(t, bindingConfig(bindingValues{}), bindingRaw(state), attachedPolicies("pol-1"))
	if !got.Equal(attachedPolicies("pol-1")) {
		t.Fatalf("omitted policies_ids must carry state, got %v", got)
	}
}

func TestPoliciesIDsModifier_DefaultPoliciesForcesUnknown(t *testing.T) {
	config := bindingConfig(bindingValues{defaultPols: boolPtr(true)})
	got := runPoliciesModifier(t, config, bindingRaw(bindingValues{policies: []string{"pol-1"}}), attachedPolicies("pol-1"))
	if !got.IsUnknown() {
		t.Fatalf("switching to default_policies must replan policies_ids as unknown, got %v", got)
	}
}

func TestPoliciesIDsModifier_ProjectIDForcesUnknown(t *testing.T) {
	config := bindingConfig(bindingValues{projectID: strPtr("proj-1")})
	got := runPoliciesModifier(t, config, bindingRaw(bindingValues{policies: []string{"pol-1"}}), attachedPolicies("pol-1"))
	if !got.IsUnknown() {
		t.Fatalf("switching to project_id must replan policies_ids as unknown, got %v", got)
	}
}

func TestPoliciesIDsModifier_NoAttachedPoliciesCarriesState(t *testing.T) {
	// Nothing to clear: state has no policies, so even a default_policies switch keeps state.
	config := bindingConfig(bindingValues{defaultPols: boolPtr(true)})
	got := runPoliciesModifier(t, config, bindingRaw(bindingValues{}), types.SetNull(types.StringType))
	if !got.IsNull() {
		t.Fatalf("empty prior policies must carry (null), got %v", got)
	}
}

func TestPoliciesIDsModifier_CreateLeavesPlanAlone(t *testing.T) {
	req := planmodifier.SetRequest{
		Config:      bindingConfig(bindingValues{}),
		ConfigValue: types.SetNull(types.StringType),
		State:       bindingState(nullBindingRaw()),
		StateValue:  types.SetNull(types.StringType),
		PlanValue:   types.SetUnknown(types.StringType),
	}
	resp := planmodifier.SetResponse{PlanValue: req.PlanValue}
	PoliciesIDsPlanModifier().PlanModifySet(context.Background(), req, &resp)
	if !resp.PlanValue.IsUnknown() {
		t.Fatalf("create must not rewrite the plan, got %v", resp.PlanValue)
	}
}

func TestPoliciesIDsModifier_ExplicitConfigLeavesPlanAlone(t *testing.T) {
	req := planmodifier.SetRequest{
		Config:      bindingConfig(bindingValues{policies: []string{"pol-2"}}),
		ConfigValue: attachedPolicies("pol-2"),
		State:       bindingState(bindingRaw(bindingValues{policies: []string{"pol-1"}})),
		StateValue:  attachedPolicies("pol-1"),
		PlanValue:   attachedPolicies("pol-2"),
	}
	resp := planmodifier.SetResponse{PlanValue: req.PlanValue}
	PoliciesIDsPlanModifier().PlanModifySet(context.Background(), req, &resp)
	if !resp.PlanValue.Equal(attachedPolicies("pol-2")) {
		t.Fatalf("explicit config must win untouched, got %v", resp.PlanValue)
	}
}

// --- default_policies ---

func runDefaultPoliciesModifier(t *testing.T, config tfsdk.Config, stateRaw tftypes.Value, stateValue types.Bool) types.Bool {
	t.Helper()
	req := planmodifier.BoolRequest{
		Config:      config,
		ConfigValue: types.BoolNull(),
		State:       bindingState(stateRaw),
		StateValue:  stateValue,
		PlanValue:   types.BoolUnknown(),
	}
	resp := planmodifier.BoolResponse{PlanValue: req.PlanValue}
	DefaultPoliciesPlanModifier().PlanModifyBool(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	return resp.PlanValue
}

func TestDefaultPoliciesModifier_OmittedCarriesState(t *testing.T) {
	got := runDefaultPoliciesModifier(t, bindingConfig(bindingValues{}), bindingRaw(bindingValues{defaultPols: boolPtr(true)}), types.BoolValue(true))
	if got.IsUnknown() || !got.ValueBool() {
		t.Fatalf("omitted default_policies must carry state true, got %v", got)
	}
}

func TestDefaultPoliciesModifier_PoliciesSwitchForcesUnknown(t *testing.T) {
	config := bindingConfig(bindingValues{policies: []string{"pol-1"}})
	got := runDefaultPoliciesModifier(t, config, bindingRaw(bindingValues{defaultPols: boolPtr(true)}), types.BoolValue(true))
	if !got.IsUnknown() {
		t.Fatalf("attaching policies while state says true must replan as unknown, got %v", got)
	}
}

func TestDefaultPoliciesModifier_ProjectSwitchForcesUnknown(t *testing.T) {
	config := bindingConfig(bindingValues{projectID: strPtr("proj-1")})
	got := runDefaultPoliciesModifier(t, config, bindingRaw(bindingValues{defaultPols: boolPtr(true)}), types.BoolValue(true))
	if !got.IsUnknown() {
		t.Fatalf("binding a project while state says true must replan as unknown, got %v", got)
	}
}

func TestDefaultPoliciesModifier_AlreadyFalseSettles(t *testing.T) {
	// Once the derived flag is false it can only stay false — no forced re-read.
	config := bindingConfig(bindingValues{policies: []string{"pol-1"}})
	got := runDefaultPoliciesModifier(t, config, bindingRaw(bindingValues{defaultPols: boolPtr(false)}), types.BoolValue(false))
	if got.IsUnknown() || got.ValueBool() {
		t.Fatalf("false prior state must settle to false, got %v", got)
	}
}

// --- project_id ---

func runProjectIDModifier(t *testing.T, config tfsdk.Config, stateRaw tftypes.Value, stateValue types.String) types.String {
	t.Helper()
	req := planmodifier.StringRequest{
		Config:      config,
		ConfigValue: types.StringNull(),
		State:       bindingState(stateRaw),
		StateValue:  stateValue,
		PlanValue:   types.StringUnknown(),
	}
	resp := planmodifier.StringResponse{PlanValue: req.PlanValue}
	ProjectIDPlanModifier().PlanModifyString(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	return resp.PlanValue
}

func TestProjectIDModifier_OmittedCarriesState(t *testing.T) {
	got := runProjectIDModifier(t, bindingConfig(bindingValues{}), bindingRaw(bindingValues{projectID: strPtr("proj-1")}), types.StringValue("proj-1"))
	if got.IsUnknown() || got.ValueString() != "proj-1" {
		t.Fatalf("omitted project_id must carry state, got %v", got)
	}
}

func TestProjectIDModifier_PoliciesSwitchForcesUnknown(t *testing.T) {
	config := bindingConfig(bindingValues{policies: []string{"pol-1"}})
	got := runProjectIDModifier(t, config, bindingRaw(bindingValues{projectID: strPtr("proj-1")}), types.StringValue("proj-1"))
	if !got.IsUnknown() {
		t.Fatalf("switching to policies must replan project_id as unknown, got %v", got)
	}
}

func TestProjectIDModifier_NoPriorProjectCarriesState(t *testing.T) {
	config := bindingConfig(bindingValues{policies: []string{"pol-1"}})
	got := runProjectIDModifier(t, config, bindingRaw(bindingValues{}), types.StringNull())
	if !got.IsNull() {
		t.Fatalf("no prior project binding: nothing to clear, must carry null, got %v", got)
	}
}

func TestProjectIDModifier_CreateLeavesPlanAlone(t *testing.T) {
	req := planmodifier.StringRequest{
		Config:      bindingConfig(bindingValues{}),
		ConfigValue: types.StringNull(),
		State:       bindingState(nullBindingRaw()),
		StateValue:  types.StringNull(),
		PlanValue:   types.StringUnknown(),
	}
	resp := planmodifier.StringResponse{PlanValue: req.PlanValue}
	ProjectIDPlanModifier().PlanModifyString(context.Background(), req, &resp)
	if !resp.PlanValue.IsUnknown() {
		t.Fatalf("create must not rewrite the plan, got %v", resp.PlanValue)
	}
}
