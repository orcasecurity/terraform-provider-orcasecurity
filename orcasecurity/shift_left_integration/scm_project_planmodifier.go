package shift_left_integration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// UseStateForUnknown semantics unless config switches from project to policies.
type projectIDPlanModifier struct{}

func ProjectIDPlanModifier() planmodifier.String { return projectIDPlanModifier{} }

func (projectIDPlanModifier) Description(context.Context) string {
	return "Preserves the existing project binding when project_id is omitted, unless the config switches to policies."
}

func (m projectIDPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (projectIDPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	// No prior state to carry forward yet (create): leave the core-proposed value alone.
	if req.State.Raw.IsNull() {
		return
	}

	var policies types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("policies_ids"), &policies)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if hasExplicitPolicies(policies) && !req.StateValue.IsNull() && req.StateValue.ValueString() != "" {
		resp.PlanValue = types.StringUnknown()
		return
	}

	resp.PlanValue = req.StateValue
}
