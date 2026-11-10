package shift_left_integration

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type policiesIDsPlanModifier struct{}

func PoliciesIDsPlanModifier() planmodifier.Set { return policiesIDsPlanModifier{} }

func (policiesIDsPlanModifier) Description(context.Context) string {
	return "Preserves attached policy IDs when policies_ids is omitted, unless the config switches to default_policies or project_id."
}

func (m policiesIDsPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (policiesIDsPlanModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if req.State.Raw.IsNull() {
		return
	}

	var defaultPolicies types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("default_policies"), &defaultPolicies)...)
	if resp.Diagnostics.HasError() {
		return
	}
	stateHasPolicies := !req.StateValue.IsNull() && len(req.StateValue.Elements()) > 0

	var projectID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project_id"), &projectID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := ""
	if tfconv.Known(projectID) {
		project = projectID.ValueString()
	}

	if policiesIDsShouldClear(tfconv.BoolIsTrue(defaultPolicies), project, stateHasPolicies) {
		resp.PlanValue = types.SetUnknown(types.StringType)
		return
	}
	resp.PlanValue = req.StateValue
}

func policiesIDsShouldClear(defaultPoliciesTrue bool, configProjectID string, stateHasPolicies bool) bool {
	if !stateHasPolicies {
		return false
	}
	return defaultPoliciesTrue || configProjectID != ""
}

// Do not carry true forward when config switches to policies_ids or project_id.
type defaultPoliciesPlanModifier struct{}

func DefaultPoliciesPlanModifier() planmodifier.Bool { return defaultPoliciesPlanModifier{} }

func (defaultPoliciesPlanModifier) Description(context.Context) string {
	return "Preserves default_policies when omitted, unless the config switches to policies_ids or project_id."
}

func (m defaultPoliciesPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (defaultPoliciesPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if req.State.Raw.IsNull() {
		return
	}

	var policies types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("policies_ids"), &policies)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var projectID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project_id"), &projectID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	project := ""
	if tfconv.Known(projectID) {
		project = projectID.ValueString()
	}

	// Config is switching to a shape that makes the derived flag false. Only
	// force unknown while prior state still says true — once false, settle.
	if (hasExplicitPolicies(policies) || project != "") &&
		!req.StateValue.IsNull() && !req.StateValue.IsUnknown() && req.StateValue.ValueBool() {
		resp.PlanValue = types.BoolUnknown()
		return
	}

	resp.PlanValue = req.StateValue
}
