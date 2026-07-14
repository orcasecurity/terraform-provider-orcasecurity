package admission_controller

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// stringListFromAPI maps an API string list onto state. An empty API list with
// a null prior stays null (config omitted the attribute); otherwise the API
// value wins.
func stringListFromAPI(ctx context.Context, prior types.List, values []string) (types.List, diag.Diagnostics) {
	if len(values) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, values)
}

func stringListToSlice(ctx context.Context, list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return []string{}
	}
	var out []string
	list.ElementsAs(ctx, &out, false)
	return out
}
