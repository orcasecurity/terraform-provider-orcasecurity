package compliance_framework

import (
	"context"
	"sort"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const maxCatalogDepth = 3

type frameworkModel struct {
	ID                         types.String `tfsdk:"id"`
	DisplayName                types.String `tfsdk:"display_name"`
	Description                types.String `tfsdk:"description"`
	Custom                     types.Bool   `tfsdk:"custom"`
	Active                     types.Bool   `tfsdk:"active"`
	SelectionScopes            types.List   `tfsdk:"selection_scopes"`
	Type                       types.String `tfsdk:"type"`
	Version                    types.String `tfsdk:"version"`
	VersionAgnosticDisplayName types.String `tfsdk:"version_agnostic_display_name"`
	IsReady                    types.Bool   `tfsdk:"is_ready"`
	FrameworkCloudVendors      types.List   `tfsdk:"framework_cloud_vendors"`
	IconFamily                 types.String `tfsdk:"icon_family"`
	OrcaEndOfSupportDate       types.String `tfsdk:"orca_end_of_support_date"`
	Visibility                 types.String `tfsdk:"visibility"`
	OriginType                 types.String `tfsdk:"origin_type"`
	CreatedAt                  types.String `tfsdk:"created_at"`
	UpdatedAt                  types.String `tfsdk:"updated_at"`
	CreatedBy                  types.String `tfsdk:"created_by"`
	UpdatedBy                  types.String `tfsdk:"updated_by"`
	IsForcedCloudVendors       types.Bool   `tfsdk:"is_forced_cloud_vendors"`
}

type frameworkFilters struct {
	custom      types.Bool
	active      types.Bool
	typ         types.String
	displayName types.String
	search      types.String
}

func catalogObjectTypeFromAttributes(attrs map[string]schema.Attribute) types.ObjectType {
	m := make(map[string]attr.Type, len(attrs))
	for k, a := range attrs {
		m[k] = a.GetType()
	}
	return types.ObjectType{AttrTypes: m}
}

func catalogSectionObjectType(remainingDepth int) types.ObjectType {
	return catalogObjectTypeFromAttributes(catalogSectionAttributes(remainingDepth))
}

func catalogTestObjectType() types.ObjectType {
	return catalogObjectTypeFromAttributes(catalogTestAttributes())
}

func frameworkToModel(ctx context.Context, fw api_client.ComplianceFramework) (frameworkModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	sorted := append([]string(nil), fw.SelectionScopes...)
	sort.Strings(sorted)
	scopes, d := tfconv.StringListFromAPI(ctx, sorted)
	diags.Append(d...)
	vendors, d := tfconv.StringListFromAPI(ctx, fw.FrameworkCloudVendors)
	diags.Append(d...)
	return frameworkModel{
		ID:                         types.StringValue(fw.ID),
		DisplayName:                types.StringValue(fw.DisplayName),
		Description:                tfconv.StringPtrOrNull(fw.Description),
		Custom:                     types.BoolValue(fw.Custom),
		Active:                     types.BoolValue(fw.Active),
		SelectionScopes:            scopes,
		Type:                       tfconv.StringPtrOrNull(fw.Type),
		Version:                    tfconv.StringPtrOrNull(fw.Version),
		VersionAgnosticDisplayName: tfconv.StringPtrOrNull(fw.VersionAgnosticDisplayName),
		IsReady:                    tfconv.BoolPtrOrNull(fw.IsReady),
		FrameworkCloudVendors:      vendors,
		IconFamily:                 tfconv.StringPtrOrNull(fw.IconFamily),
		OrcaEndOfSupportDate:       tfconv.StringPtrOrNull(fw.OrcaEndOfSupportDate),
		Visibility:                 tfconv.StringPtrOrNull(fw.Visibility),
		OriginType:                 tfconv.StringPtrOrNull(fw.OriginType),
		CreatedAt:                  tfconv.StringPtrOrNull(fw.CreatedAt),
		UpdatedAt:                  tfconv.StringPtrOrNull(fw.UpdatedAt),
		CreatedBy:                  tfconv.StringPtrOrNull(fw.CreatedBy),
		UpdatedBy:                  tfconv.StringPtrOrNull(fw.UpdatedBy),
		IsForcedCloudVendors:       tfconv.BoolPtrOrNull(fw.IsForcedCloudVendors),
	}, diags
}

func matchBoolFilter(filter types.Bool, got bool) bool {
	return filter.IsNull() || filter.IsUnknown() || got == filter.ValueBool()
}

func stringFilterValue(v types.String) (string, bool) {
	if v.IsNull() || v.IsUnknown() {
		return "", false
	}
	s := v.ValueString()
	return s, s != ""
}

func searchMatches(fw api_client.ComplianceFramework, q string) bool {
	q = strings.ToLower(q)
	if strings.Contains(strings.ToLower(fw.DisplayName), q) {
		return true
	}
	if fw.Description == nil {
		return false
	}
	return strings.Contains(strings.ToLower(*fw.Description), q)
}

func matchFramework(fw api_client.ComplianceFramework, f frameworkFilters) bool {
	if !matchBoolFilter(f.custom, fw.Custom) {
		return false
	}
	if !matchBoolFilter(f.active, fw.Active) {
		return false
	}
	if want, ok := stringFilterValue(f.typ); ok && (fw.Type == nil || *fw.Type != want) {
		return false
	}
	if want, ok := stringFilterValue(f.displayName); ok && fw.DisplayName != want {
		return false
	}
	if q, ok := stringFilterValue(f.search); ok && !searchMatches(fw, q) {
		return false
	}
	return true
}

func filterAndSort(ctx context.Context, all map[string]api_client.ComplianceFramework, f frameworkFilters) ([]frameworkModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	ids := make([]string, 0, len(all))
	for id, fw := range all {
		if matchFramework(fw, f) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := make([]frameworkModel, 0, len(ids))
	for _, id := range ids {
		m, d := frameworkToModel(ctx, all[id])
		diags.Append(d...)
		out = append(out, m)
	}
	return out, diags
}

func catalogTestsToModel(ctx context.Context, tests []api_client.ComplianceCatalogTest) (types.List, diag.Diagnostics) {
	elem := catalogTestObjectType()
	if len(tests) == 0 {
		return types.ListNull(elem), nil
	}
	var diags diag.Diagnostics
	vals := make([]attr.Value, len(tests))
	for i, t := range tests {
		vendors, d := tfconv.StringListFromAPI(ctx, t.CloudVendors)
		diags.Append(d...)
		obj, d := types.ObjectValue(elem.AttrTypes, map[string]attr.Value{
			"name":                tfconv.StringOrNull(t.Name),
			"rule_id":             types.StringValue(t.RuleID),
			"reference_id":        tfconv.StringOrNull(t.ReferenceID),
			"origin_framework_id": tfconv.StringOrNull(t.OriginFrameworkID),
			"cloud_vendors":       vendors,
			"control_unique_id":   tfconv.StringOrNull(t.ControlUniqueID),
			"priority":            tfconv.StringOrNull(t.Priority),
			"cis_level":           tfconv.StringOrNull(string(t.CISLevel)),
		})
		diags.Append(d...)
		vals[i] = obj
	}
	list, d := types.ListValue(elem, vals)
	diags.Append(d...)
	return list, diags
}

func catalogSectionsToModel(ctx context.Context, sections []api_client.ComplianceCatalogSection, remainingDepth int) (types.List, diag.Diagnostics) {
	elem := catalogSectionObjectType(remainingDepth)
	if len(sections) == 0 {
		return types.ListNull(elem), nil
	}
	var diags diag.Diagnostics
	vals := make([]attr.Value, len(sections))
	for i, s := range sections {
		tests, d := catalogTestsToModel(ctx, s.Tests)
		diags.Append(d...)
		attrs := map[string]attr.Value{
			"name":  types.StringValue(s.Name),
			"tests": tests,
		}
		if remainingDepth > 0 {
			nested, d := catalogSectionsToModel(ctx, s.Sections, remainingDepth-1)
			diags.Append(d...)
			attrs["sections"] = nested
		}
		obj, d := types.ObjectValue(elem.AttrTypes, attrs)
		diags.Append(d...)
		vals[i] = obj
	}
	list, d := types.ListValue(elem, vals)
	diags.Append(d...)
	return list, diags
}
