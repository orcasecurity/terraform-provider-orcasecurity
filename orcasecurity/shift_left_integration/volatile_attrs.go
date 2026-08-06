package shift_left_integration

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Server-owned Computed attrs: carry forward on no-op plans (TF <1.4), leave unknown when a write is pending.
//
// Settled by ModifyPlan; exempt from per-attribute UseStateForUnknown in schema settle tests.
func ScmVolatileAttrNames() []string {
	return []string{"integration_status", "scan_all_state", "integrated_repositories_count", "scm_posture_policy_id"}
}

// Name must be in ScmVolatileAttrNames.
func ComputedVolatileString(description string) rschema.StringAttribute {
	return rschema.StringAttribute{Computed: true, Description: description}
}

func ComputedVolatileInt64(description string) rschema.Int64Attribute {
	return rschema.Int64Attribute{Computed: true, Description: description}
}

// After attribute modifiers: any non-volatile plan≠state means the apply will write.
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

// Unknown plan values count as a write (unresolved refs cannot be proven no-op).
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
