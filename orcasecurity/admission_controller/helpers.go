package admission_controller

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stringFromAPI maps an optional API string onto state. The API echoes unset
// descriptions as null or "", so when the prior state (or import: null prior)
// holds no value and the API agrees there is none, keep null to avoid a
// perpetual `"" != null` diff. A user-configured explicit "" is preserved.
func stringFromAPI(prior types.String, value *string) types.String {
	if value == nil || *value == "" {
		if !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == "" {
			return prior // user explicitly configured ""
		}
		return types.StringNull()
	}
	return types.StringValue(*value)
}
