package shift_left_integration

import (
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// Reject default_policies combos that cannot round-trip; call after plan modifiers.
func ValidateScmBindingPlan(f *ScmConfigFields, diags *diag.Diagnostics) {
	if f == nil {
		return
	}

	// C1: explicit built-ins + explicit policy list.
	if tfconv.BoolIsTrue(f.DefaultPolicies) && tfconv.Known(f.PoliciesIds) {
		diags.AddAttributeError(
			path.Root("policies_ids"),
			"Conflicting policy binding",
			"default_policies = true and policies_ids cannot be set together. "+
				"default_policies attaches every Orca built-in policy and clears any explicit list; "+
				"set only one of them.",
		)
	}

	// C2: default_policies = false with neither policies nor a project.
	// Unknowns are deferred (create-time unknowns, or plan modifiers still settling).
	if !tfconv.Known(f.DefaultPolicies) || f.DefaultPolicies.ValueBool() {
		return
	}
	if f.PoliciesIds.IsUnknown() || f.ProjectID.IsUnknown() {
		return
	}
	hasPolicies := tfconv.Known(f.PoliciesIds) && len(f.PoliciesIds.Elements()) > 0
	hasProject := tfconv.Known(f.ProjectID) && f.ProjectID.ValueString() != ""
	if hasPolicies || hasProject {
		return
	}
	diags.AddAttributeError(
		path.Root("default_policies"),
		"default_policies = false requires policies or a project",
		"The API derives default_policies on read from whether the unit has attached policies or a project. "+
			"Setting default_policies = false with neither policies_ids nor project_id can never round-trip: "+
			"the next read always returns true. Attach policies_ids, bind project_id, or set default_policies = true.",
	)
}
