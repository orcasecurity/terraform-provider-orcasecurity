package shift_left_integration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// installationModePlanModifier is UseStateForUnknown, except that a legacy SCAN_ALL
// held in state plans as the value the write path will actually send.
//
// Read reports installation_mode verbatim so a unit still on the legacy SCAN_ALL is
// visible rather than silently displayed as SELECTED_REPOSITORIES. Import performs no
// write, so that is the value that lands in state. Carrying it forward unchanged would
// promise a planned value the API rejects: the write remaps SCAN_ALL, the read-back
// returns SELECTED_REPOSITORIES, and the apply fails with "inconsistent result after
// apply". Planning the remapped value instead keeps the apply consistent and shows the
// migration in the plan, which is what the apply genuinely does.
type installationModePlanModifier struct{}

func InstallationModePlanModifier() planmodifier.String { return installationModePlanModifier{} }

func (installationModePlanModifier) Description(context.Context) string {
	return "Carries installation_mode forward, migrating a legacy SCAN_ALL to the value the API will store."
}

func (m installationModePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (installationModePlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Configured explicitly, no prior state (create), or already resolved: leave it alone.
	if !req.ConfigValue.IsNull() || req.State.Raw.IsNull() || !req.PlanValue.IsUnknown() {
		return
	}
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	resp.PlanValue = types.StringValue(normalizeInstallationMode(req.StateValue.ValueString()))
}
