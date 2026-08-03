package shift_left_integration

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Volatile attributes are Computed and server-owned: the API may move them as a
// side effect of any write (repo enrollment, health flaps, scan-all progress).
// They carry no attribute plan modifiers. AdoptedUnitOps.ModifyPlan settles them
// via settleVolatileAttrs after all attribute plan modifiers have run: carried
// forward from state when the plan writes nothing (so empty plans settle on
// Terraform 1.0–1.3, where core replans Computed attributes as unknown), left
// unknown when any write is pending (so a side-effecting write cannot produce
// an inconsistent apply result).

// ScmVolatileAttrNames lists the volatile attributes settled by
// AdoptedUnitOps.ModifyPlan. Exported for the schema settle test, which exempts
// them from the per-attribute carry-forward modifier requirement.
func ScmVolatileAttrNames() []string {
	return []string{"integration_status", "scan_all_state", "integrated_repositories_count", "scm_posture_policy_id"}
}

// ComputedVolatileString declares a volatile Computed string attribute; the
// name must be in ScmVolatileAttrNames or it will never settle on TF 1.0–1.3.
func ComputedVolatileString(description string) rschema.StringAttribute {
	return rschema.StringAttribute{Computed: true, Description: description}
}

// ComputedVolatileInt64 is the int64 counterpart of ComputedVolatileString.
func ComputedVolatileInt64(description string) rschema.Int64Attribute {
	return rschema.Int64Attribute{Computed: true, Description: description}
}

// settleVolatileAttrs decides "will this plan write?" from the already-modified
// plan. By the time the resource-level ModifyPlan runs, attribute plan modifiers
// have hydrated every omitted writable from state (UseStateForUnknown and the
// binding modifiers), deliberately left a writable unknown only when the config
// is switching bindings (a genuine write), and rewritten a legacy
// installation_mode to the value the write path will send. So "some non-volatile
// attribute differs from state" is exactly "this apply will call the API".
func settleVolatileAttrs(req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || resp.Plan.Raw.IsNull() {
		return // create leaves volatiles unknown; destroy has no plan to settle
	}
	var planObj, stateObj map[string]tftypes.Value
	if resp.Plan.Raw.As(&planObj) != nil || req.State.Raw.As(&stateObj) != nil {
		return
	}

	writePending := scmWritePending(planObj, stateObj)
	for _, name := range ScmVolatileAttrNames() {
		pv, ok := planObj[name]
		if !ok {
			continue
		}
		if writePending {
			planObj[name] = tftypes.NewValue(pv.Type(), tftypes.UnknownValue)
		} else {
			planObj[name] = stateObj[name]
		}
	}
	resp.Plan.Raw = tftypes.NewValue(resp.Plan.Raw.Type(), planObj)
}

// scmWritePending reports whether the plan differs from state on any
// non-volatile attribute. State is always fully known, so the deep Equal also
// fails when a plan value is (or contains) an unknown — an unresolved
// cross-resource reference cannot be proven a no-op and counts as a write.
func scmWritePending(planObj, stateObj map[string]tftypes.Value) bool {
	volatile := map[string]bool{}
	for _, name := range ScmVolatileAttrNames() {
		volatile[name] = true
	}
	for name, pv := range planObj {
		if volatile[name] {
			continue
		}
		sv, ok := stateObj[name]
		if !ok || !pv.Equal(sv) {
			return true
		}
	}
	return false
}
