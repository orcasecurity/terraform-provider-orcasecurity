// compliance_frameworks value plumbing shared by the custom alert resources.
package alert_common

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

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

// FrameworksRequest converts a plan or state value into the API payload. A null
// or unknown list yields no frameworks, so a config that says nothing about the
// links never sends a clearing request.
func FrameworksRequest(ctx context.Context, list types.List) ([]api_client.AlertComplianceFramework, diag.Diagnostics) {
	frameworks, diags := FrameworksFromList(ctx, list)
	if diags.HasError() {
		return nil, diags
	}

	var request []api_client.AlertComplianceFramework
	for _, framework := range frameworks {
		category, subCategory, subSubCategory := api_client.SplitComplianceSection(framework.Section.ValueString())
		request = append(request, api_client.AlertComplianceFramework{
			Name:           framework.Name.ValueString(),
			Category:       category,
			SubCategory:    subCategory,
			SubSubCategory: subSubCategory,
			Priority:       framework.Priority.ValueString(),
		})
	}
	return request, diags
}

// FrameworksState converts read-back frameworks into the state value.
func FrameworksState(ctx context.Context, frameworks []api_client.AlertComplianceFramework) (types.List, diag.Diagnostics) {
	values := make([]Framework, 0, len(frameworks))
	for _, framework := range frameworks {
		values = append(values, Framework{
			Name: types.StringValue(framework.Name),
			Section: types.StringValue(api_client.JoinComplianceSection(
				framework.Category, framework.SubCategory, framework.SubSubCategory)),
			Priority: types.StringValue(framework.Priority),
		})
	}
	return FrameworksToList(ctx, values)
}

// FrameworksAfterCreate resolves the value a create writes to state. An
// Optional+Computed list is unknown when the config does not declare it, and
// the links the alert actually ended up with are whatever the API reports.
// A declared value is kept as planned, so the state matches the plan exactly.
func FrameworksAfterCreate(ctx context.Context, planned types.List, frameworks []api_client.AlertComplianceFramework) (types.List, diag.Diagnostics) {
	if !planned.IsUnknown() {
		return planned, nil
	}
	return FrameworksState(ctx, frameworks)
}
