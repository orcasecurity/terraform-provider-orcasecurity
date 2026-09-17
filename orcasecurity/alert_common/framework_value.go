// compliance_frameworks value plumbing shared by the custom alert resources.
package alert_common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Framework is one compliance_frameworks element, independent of which alert
// API type it is converted to.
type Framework struct {
	Name     types.String `tfsdk:"name"`
	Section  types.String `tfsdk:"section"`
	Priority types.String `tfsdk:"priority"`
}

// FrameworkObjectType is the element type of the compliance_frameworks list.
func FrameworkObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":     types.StringType,
		"section":  types.StringType,
		"priority": types.StringType,
	}}
}

// FrameworksToList converts read-back frameworks into the state value. Alerts
// with no links get an empty list rather than null, so that a config declaring
// `compliance_frameworks = []` stays free of a permanent null-vs-empty diff.
func FrameworksToList(ctx context.Context, frameworks []Framework) (types.List, diag.Diagnostics) {
	if frameworks == nil {
		frameworks = []Framework{}
	}
	return types.ListValueFrom(ctx, FrameworkObjectType(), frameworks)
}

// FrameworksFromList reads a plan or state value. A null or unknown list means
// "the config does not speak about the links", and yields no frameworks.
func FrameworksFromList(ctx context.Context, list types.List) ([]Framework, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var frameworks []Framework
	diags := list.ElementsAs(ctx, &frameworks, false)
	return frameworks, diags
}

// FrameworksCount reports how many links a plan or state value holds.
func FrameworksCount(list types.List) int {
	if list.IsNull() || list.IsUnknown() {
		return 0
	}
	return len(list.Elements())
}
