package integrations_common

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OptionalStringMatchPlan maps an optional API string back to a Terraform optional
// string: APIs echo unset values as null or "", so when the plan/prior state holds
// no value and the API agrees there is none, keep null to avoid a perpetual
// `"" != null` diff. A user-configured explicit "" is preserved.
func OptionalStringMatchPlan(planOrPrior types.String, api *string) types.String {
	if api == nil || *api == "" {
		if !planOrPrior.IsNull() && !planOrPrior.IsUnknown() && planOrPrior.ValueString() == "" {
			return planOrPrior // user explicitly configured ""
		}
		return types.StringNull()
	}
	return types.StringValue(*api)
}
