package compliance_framework

import (
	"sort"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
}

type catalogTestModel struct {
	Name              types.String `tfsdk:"name"`
	RuleID            types.String `tfsdk:"rule_id"`
	ReferenceID       types.String `tfsdk:"reference_id"`
	OriginFrameworkID types.String `tfsdk:"origin_framework_id"`
	CloudVendors      types.List   `tfsdk:"cloud_vendors"`
	ControlUniqueID   types.String `tfsdk:"control_unique_id"`
	Priority          types.String `tfsdk:"priority"`
}

type catalogLeafSectionModel struct {
	Name  types.String       `tfsdk:"name"`
	Tests []catalogTestModel `tfsdk:"tests"`
}

type catalogMidSectionModel struct {
	Name     types.String              `tfsdk:"name"`
	Tests    []catalogTestModel        `tfsdk:"tests"`
	Sections []catalogLeafSectionModel `tfsdk:"sections"`
}

type catalogSectionModel struct {
	Name     types.String             `tfsdk:"name"`
	Tests    []catalogTestModel       `tfsdk:"tests"`
	Sections []catalogMidSectionModel `tfsdk:"sections"`
}

func optionalString(s *string) types.String {
	if s == nil || *s == "" {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

func optionalNonEmpty(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func optionalBool(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}

func optionalStringList(values []string) types.List {
	if values == nil {
		return types.ListNull(types.StringType)
	}
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}
	return types.ListValueMust(types.StringType, elems)
}

func sortedStringList(values []string) types.List {
	cp := append([]string(nil), values...)
	sort.Strings(cp)
	return optionalStringList(cp)
}

func frameworkToModel(fw api_client.ComplianceFramework) frameworkModel {
	return frameworkModel{
		ID:                         types.StringValue(fw.ID),
		DisplayName:                types.StringValue(fw.DisplayName),
		Description:                optionalString(fw.Description),
		Custom:                     types.BoolValue(fw.Custom),
		Active:                     types.BoolValue(fw.Active),
		SelectionScopes:            sortedStringList(fw.SelectionScopes),
		Type:                       optionalString(fw.Type),
		Version:                    optionalString(fw.Version),
		VersionAgnosticDisplayName: optionalString(fw.VersionAgnosticDisplayName),
		IsReady:                    optionalBool(fw.IsReady),
		FrameworkCloudVendors:      optionalStringList(fw.FrameworkCloudVendors),
		IconFamily:                 optionalString(fw.IconFamily),
		OrcaEndOfSupportDate:       optionalString(fw.OrcaEndOfSupportDate),
		Visibility:                 optionalString(fw.Visibility),
	}
}

type frameworkFilters struct {
	custom      types.Bool
	active      types.Bool
	typ         types.String
	displayName types.String
	search      types.String
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

func pointerEquals(got *string, want string) bool {
	return got != nil && *got == want
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
	if want, ok := stringFilterValue(f.typ); ok && !pointerEquals(fw.Type, want) {
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

func filterAndSort(all map[string]api_client.ComplianceFramework, f frameworkFilters) []frameworkModel {
	ids := make([]string, 0, len(all))
	for id, fw := range all {
		if fw.ID == "" {
			fw.ID = id
		}
		if matchFramework(fw, f) {
			ids = append(ids, fw.ID)
		}
	}
	sort.Strings(ids)
	out := make([]frameworkModel, 0, len(ids))
	for _, id := range ids {
		fw := all[id]
		if fw.ID == "" {
			fw.ID = id
		}
		out = append(out, frameworkToModel(fw))
	}
	return out
}

func catalogTestsToModel(tests []api_client.ComplianceCatalogTest) []catalogTestModel {
	if len(tests) == 0 {
		return nil
	}
	out := make([]catalogTestModel, len(tests))
	for i, t := range tests {
		out[i] = catalogTestModel{
			Name:              optionalNonEmpty(t.Name),
			RuleID:            types.StringValue(t.RuleID),
			ReferenceID:       optionalNonEmpty(t.ReferenceID),
			OriginFrameworkID: optionalNonEmpty(t.OriginFrameworkID),
			CloudVendors:      optionalStringList(t.CloudVendors),
			ControlUniqueID:   optionalNonEmpty(t.ControlUniqueID),
			Priority:          optionalNonEmpty(t.Priority),
		}
	}
	return out
}

func catalogSectionsToModel(sections []api_client.ComplianceCatalogSection) []catalogSectionModel {
	if len(sections) == 0 {
		return nil
	}
	out := make([]catalogSectionModel, len(sections))
	for i, s := range sections {
		out[i] = catalogSectionModel{
			Name:  types.StringValue(s.Name),
			Tests: catalogTestsToModel(s.Tests),
		}
		if len(s.Sections) == 0 {
			continue
		}
		mids := make([]catalogMidSectionModel, len(s.Sections))
		for j, mid := range s.Sections {
			mids[j] = catalogMidSectionModel{
				Name:  types.StringValue(mid.Name),
				Tests: catalogTestsToModel(mid.Tests),
			}
			if len(mid.Sections) == 0 {
				continue
			}
			leaves := make([]catalogLeafSectionModel, len(mid.Sections))
			for k, leaf := range mid.Sections {
				leaves[k] = catalogLeafSectionModel{
					Name:  types.StringValue(leaf.Name),
					Tests: catalogTestsToModel(leaf.Tests),
				}
			}
			mids[j].Sections = leaves
		}
		out[i].Sections = mids
	}
	return out
}
