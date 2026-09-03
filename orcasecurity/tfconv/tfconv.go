package tfconv

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func Known(v attr.Value) bool {
	return !v.IsNull() && !v.IsUnknown()
}

func BoolIsTrue(b types.Bool) bool {
	return Known(b) && b.ValueBool()
}

func StringIsSet(s types.String) bool {
	return Known(s) && s.ValueString() != ""
}

func stringElements(elems []attr.Value) []string {
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		if v, ok := e.(types.String); ok && Known(v) {
			out = append(out, v.ValueString())
		}
	}
	return out
}

func stringValues(values []string) []attr.Value {
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}
	return elems
}

// Null and unknown sets become nil (omitted from the JSON payload).
func SetToStringSlice(s types.Set) []string {
	if !Known(s) {
		return nil
	}
	return stringElements(s.Elements())
}

// Never returns nil — send [] to clear on PATCH endpoints where omitted keys are unchanged.
func SetToStringSliceNonNull(s types.Set) []string {
	if out := SetToStringSlice(s); out != nil {
		return out
	}
	return []string{}
}

// Null and unknown lists become nil (omitted from the JSON payload).
func ListToStringSlice(l types.List) []string {
	if !Known(l) {
		return nil
	}
	return stringElements(l.Elements())
}

// Never returns nil — send [] to clear on PATCH endpoints where omitted keys are unchanged.
func ListToStringSliceNonNull(l types.List) []string {
	if out := ListToStringSlice(l); out != nil {
		return out
	}
	return []string{}
}

func StringSliceToSet(values []string) types.Set {
	if len(values) == 0 {
		return types.SetNull(types.StringType)
	}
	return types.SetValueMust(types.StringType, stringValues(values))
}

// Preserve null-vs-[] so a configured empty set does not drift when the API omits values.
func StringSliceToSetPreserveNull(prior types.Set, values []string) types.Set {
	if len(values) == 0 && prior.IsNull() {
		return types.SetNull(types.StringType)
	}
	return types.SetValueMust(types.StringType, stringValues(values))
}

func StringSliceToListPreserveNull(prior types.List, values []string) types.List {
	if len(values) == 0 && prior.IsNull() {
		return types.ListNull(types.StringType)
	}
	return types.ListValueMust(types.StringType, stringValues(values))
}

func StringOrNull(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

// StringPtrOrNull maps an optional API string pointer: nil or empty becomes null.
func StringPtrOrNull(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return StringOrNull(*v)
}

// BoolPtrOrNull maps an optional API bool pointer: nil becomes null.
func BoolPtrOrNull(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

// StringListFromAPI maps an API string slice to a list. nil becomes null;
// an empty slice becomes an empty list (not null).
func StringListFromAPI(ctx context.Context, values []string) (types.List, diag.Diagnostics) {
	if values == nil {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, values)
}

// StringSetFromAPIPreserveNull maps an API string slice back to a set in
// state, preserving null when the API returns empty and the prior state was
// null.
func StringSetFromAPIPreserveNull(ctx context.Context, prior types.Set, values []string) (types.Set, diag.Diagnostics) {
	if len(values) == 0 {
		if prior.IsNull() {
			return types.SetNull(types.StringType), nil
		}
		// a nil slice would convert to a null set; the attribute was
		// configured, so state must hold an empty set, not null
		values = []string{}
	}
	return types.SetValueFrom(ctx, types.StringType, values)
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

func Int64FromAPIPtr(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}
