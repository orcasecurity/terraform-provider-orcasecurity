package shift_left_integration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// projectIDPlanModifier keeps project_id stable across plans (like
// UseStateForUnknown) while avoiding a "provider produced inconsistent result"
// error when the user switches a project-bound unit over to policies.
//
// Behavior when project_id is omitted from config:
//   - if the config now expresses policies intent (policies_ids/default_policies
//     set) and the unit is currently bound to a project, the binding will be
//     cleared on apply, so the planned value is set to unknown (any post-apply
//     value is then valid); and
//   - otherwise the prior state value is reused (no plan churn for steady
//     project-bound or policies-based units).
type projectIDPlanModifier struct{}

// ProjectIDPlanModifier returns the shared plan modifier for the project_id
// attribute of every SCM config resource.
func ProjectIDPlanModifier() planmodifier.String { return projectIDPlanModifier{} }

func (projectIDPlanModifier) Description(context.Context) string {
	return "Preserves the existing project binding when project_id is omitted, unless the config switches to policies."
}

func (m projectIDPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (projectIDPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// User set project_id explicitly: keep the config value untouched.
	if !req.ConfigValue.IsNull() {
		return
	}

	var policies types.Set
	req.Config.GetAttribute(ctx, path.Root("policies_ids"), &policies)
	var defaultPolicies types.Bool
	req.Config.GetAttribute(ctx, path.Root("default_policies"), &defaultPolicies)
	policiesIntent := (!policies.IsNull() && !policies.IsUnknown()) ||
		(!defaultPolicies.IsNull() && !defaultPolicies.IsUnknown())

	// Switching an existing project-bound unit to policies: the binding will be
	// cleared, so don't assert the old value in the plan.
	if policiesIntent && !req.StateValue.IsNull() && req.StateValue.ValueString() != "" {
		resp.PlanValue = types.StringUnknown()
		return
	}

	// Steady state: reuse prior value (UseStateForUnknown semantics).
	if !req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}
