// Package integrations_common holds helpers shared across the
// orcasecurity_integration_* resources so the per-resource files don't each
// reimplement the same Terraform Plugin Framework <-> Orca API plumbing
// (business_units round-trip, JSON-field encoding, webhook custom_headers, etc).
package integrations_common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BusinessUnitsFromAPI converts the API's business_units field into a Terraform Set value.
// When the user did not declare business_units in their config (planned value is null) and
// the API returns an empty list, preserve the null to avoid a "null vs []" diff on every
// plan.
func BusinessUnitsFromAPI(ctx context.Context, apiBus []string, planned types.Set) (types.Set, diag.Diagnostics) {
	if len(apiBus) == 0 && planned.IsNull() {
		return types.SetNull(types.StringType), nil
	}
	return types.SetValueFrom(ctx, types.StringType, apiBus)
}

// BusinessUnitsToAPI converts a Terraform Set into the []string the Orca API expects. Returns
// nil when the planned set is null or unknown so json.Marshal with “omitempty“ skips the
// field entirely (matches what the UI sends — unset, not an empty array).
func BusinessUnitsToAPI(ctx context.Context, planned types.Set, diags *diag.Diagnostics) []string {
	if planned.IsNull() || planned.IsUnknown() {
		return nil
	}
	var out []string
	diags.Append(planned.ElementsAs(ctx, &out, false)...)
	return out
}
