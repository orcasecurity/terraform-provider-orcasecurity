// Package tfconv holds the shared conversions between terraform-plugin-framework
// attribute values and the plain Go values used by api_client payloads.
package tfconv

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringListToAPI converts a types.List of strings to a Go slice.
// Null and unknown lists become nil (omitted from the JSON payload).
func StringListToAPI(ctx context.Context, list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var out []string
	_ = list.ElementsAs(ctx, &out, false)
	return out
}

// StringListToAPINonNull is StringListToAPI but never returns nil: a null,
// unknown, or empty list becomes []string{}. Use it for payloads that must
// always serialize the field as [] — e.g. partial-update endpoints where an
// omitted key keeps its remote value, so clearing a list requires sending [].
func StringListToAPINonNull(ctx context.Context, list types.List) []string {
	if out := StringListToAPI(ctx, list); out != nil {
		return out
	}
	return []string{}
}

// StringListFromAPIPreserveNull maps an API string slice back to state.
// When the API returns empty and the prior state was null (attribute not
// configured), null is preserved to avoid a perpetual null-vs-[] diff.
func StringListFromAPIPreserveNull(ctx context.Context, prior types.List, values []string) (types.List, diag.Diagnostics) {
	if len(values) == 0 && prior.IsNull() {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, values)
}

// StringOrNull maps optional API strings: empty string becomes null.
func StringOrNull(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

// Int64ToAPIPtr converts an optional types.Int64 to a pointer.
// Null and unknown values become nil (omitted from the JSON payload).
func Int64ToAPIPtr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueInt64()
	return &value
}

// Int64FromAPIPtr maps an optional API integer back to state: nil becomes null.
func Int64FromAPIPtr(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}
