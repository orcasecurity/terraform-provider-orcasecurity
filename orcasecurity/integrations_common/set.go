package integrations_common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringSliceFromSet converts a Terraform Set into a []string for the Orca API.
// A null or unknown set yields a non-nil empty slice (not nil) so callers can send
// an explicit empty array rather than omitting the field.
func StringSliceFromSet(ctx context.Context, s types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if s.IsNull() || s.IsUnknown() {
		return []string{}, diags
	}
	var out []string
	diags = s.ElementsAs(ctx, &out, false)
	return out, diags
}

// OptionalSetMatchPlan maps API slices back to Terraform optional sets: omitted config
// (null) stays null when the API returns empty, avoiding a "null vs []" diff on every plan.
func OptionalSetMatchPlan(ctx context.Context, planOrPrior types.Set, api []string) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(api) > 0 {
		return types.SetValueFrom(ctx, types.StringType, api)
	}
	if planOrPrior.IsNull() || planOrPrior.IsUnknown() {
		return types.SetNull(types.StringType), diags
	}
	return types.SetValueFrom(ctx, types.StringType, []string{})
}
