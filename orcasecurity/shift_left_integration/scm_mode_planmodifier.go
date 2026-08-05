package shift_left_integration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Carry state forward, but plan legacy SCAN_ALL as the remapped write value (also when known on TF ≥1.4).
type installationModePlanModifier struct{}

func InstallationModePlanModifier() planmodifier.String { return installationModePlanModifier{} }

func (installationModePlanModifier) Description(context.Context) string {
	return "Carries installation_mode forward, migrating a legacy SCAN_ALL to the value the API will store."
}

func (m installationModePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (installationModePlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() || req.State.Raw.IsNull() {
		return
	}
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	resp.PlanValue = types.StringValue(normalizeInstallationMode(req.StateValue.ValueString()))
}
