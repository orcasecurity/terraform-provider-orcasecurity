package integrations_common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringSliceFromList converts a Terraform List into a []string for the Orca API.
// A null or unknown list yields a non-nil empty slice (not nil) so callers can send
// an explicit empty array rather than omitting the field.
func StringSliceFromList(ctx context.Context, l types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if l.IsNull() || l.IsUnknown() {
		return []string{}, diags
	}
	var out []string
	diags = l.ElementsAs(ctx, &out, false)
	return out, diags
}

// OptionalListMatchPlan maps API slices back to Terraform optional lists: omitted config
// (null) stays null when the API returns empty, avoiding a "null vs []" diff on every plan.
func OptionalListMatchPlan(ctx context.Context, planOrPrior types.List, api []string) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(api) > 0 {
		return types.ListValueFrom(ctx, types.StringType, api)
	}
	if planOrPrior.IsNull() || planOrPrior.IsUnknown() {
		return types.ListNull(types.StringType), diags
	}
	return types.ListValueFrom(ctx, types.StringType, []string{})
}
