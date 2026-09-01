package custom_compliance_framework

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// maxSectionDepth is the API's stored depth (category / sub_category /
// sub_sub_category). The schema uses remainingDepth = maxSectionDepth-1 so a
// fourth nested `sections` is not an attribute at all.
const maxSectionDepth = 3

const mixedSectionMessage = "a section cannot have both tests and sub-sections; the API would silently flatten it and the sub-section would inherit the parent name"

const invalidSectionSummary = "Invalid section"

func objectTypeFromAttributes(attrs map[string]schema.Attribute) types.ObjectType {
	m := make(map[string]attr.Type, len(attrs))
	for k, a := range attrs {
		m[k] = a.GetType()
	}
	return types.ObjectType{AttrTypes: m}
}

func testObjectType() types.ObjectType {
	return objectTypeFromAttributes(testAttributes())
}

func sectionObjectType(remainingDepth int) types.ObjectType {
	return objectTypeFromAttributes(sectionAttributes(remainingDepth))
}

// DeriveRuleIDInFramework matches the UI: `${section.id}.${index+1}`.
func DeriveRuleIDInFramework(sectionID string, index int) string {
	return fmt.Sprintf("%s.%d", sectionID, index+1)
}

func positionalSectionID(parentID string, index int) string {
	n := fmt.Sprintf("%d", index+1)
	if parentID == "" {
		return n
	}
	return parentID + "." + n
}

func attrString(attrs map[string]attr.Value, name string) string {
	v, ok := attrs[name]
	if !ok || v.IsNull() || v.IsUnknown() {
		return ""
	}
	s, ok := v.(types.String)
	if !ok {
		return ""
	}
	return s.ValueString()
}

func asList(v attr.Value) (types.List, bool) {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return types.List{}, false
	}
	l, ok := v.(types.List)
	return l, ok
}

func listLen(v attr.Value) int {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return 0
	}
	l, ok := v.(types.List)
	if !ok {
		return 0
	}
	return len(l.Elements())
}

func testsToAPI(sectionID string, list types.List) []api_client.CustomComplianceFrameworkTest {
	if list.IsNull() || list.IsUnknown() {
		return []api_client.CustomComplianceFrameworkTest{}
	}
	out := make([]api_client.CustomComplianceFrameworkTest, 0, len(list.Elements()))
	for i, e := range list.Elements() {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		attrs := obj.Attributes()
		rid := attrString(attrs, "rule_id_in_framework")
		if rid == "" {
			rid = DeriveRuleIDInFramework(sectionID, i)
		}
		out = append(out, api_client.CustomComplianceFrameworkTest{
			RuleID:            attrString(attrs, "rule_id"),
			RuleIDInFramework: rid,
			Priority:          attrString(attrs, "priority"),
			ControlUniqueID:   attrString(attrs, "control_unique_id"),
			OriginFrameworkID: attrString(attrs, "origin_framework_id"),
		})
	}
	return out
}

func sectionsToAPI(list types.List) []api_client.CustomComplianceFrameworkSection {
	return sectionsToAPIAt(list, "")
}

func sectionsToAPIAt(list types.List, parentID string) []api_client.CustomComplianceFrameworkSection {
	if list.IsNull() || list.IsUnknown() {
		return []api_client.CustomComplianceFrameworkSection{}
	}
	out := make([]api_client.CustomComplianceFrameworkSection, 0, len(list.Elements()))
	for i, e := range list.Elements() {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		id := positionalSectionID(parentID, i)
		attrs := obj.Attributes()
		tests := []api_client.CustomComplianceFrameworkTest{}
		if l, ok := asList(attrs["tests"]); ok {
			tests = testsToAPI(id, l)
		}
		nested := []api_client.CustomComplianceFrameworkSection{}
		if l, ok := asList(attrs["sections"]); ok {
			nested = sectionsToAPIAt(l, id)
		}
		out = append(out, api_client.CustomComplianceFrameworkSection{
			Name:     attrString(attrs, "name"),
			Tests:    tests,
			Sections: nested,
		})
	}
	return out
}

func testsFromCatalog(ctx context.Context, tests []api_client.ComplianceCatalogTest) (types.List, diag.Diagnostics) {
	elem := testObjectType()
	if len(tests) == 0 {
		return types.ListNull(elem), nil
	}
	vals := make([]attr.Value, len(tests))
	var diags diag.Diagnostics
	for i, t := range tests {
		obj, d := types.ObjectValue(elem.AttrTypes, map[string]attr.Value{
			"rule_id":              types.StringValue(t.RuleID),
			"rule_id_in_framework": tfconv.StringOrNull(t.ReferenceID),
			"priority":             tfconv.StringOrNull(t.Priority),
			"control_unique_id":    tfconv.StringOrNull(t.ControlUniqueID),
			"origin_framework_id":  tfconv.StringOrNull(t.OriginFrameworkID),
		})
		diags.Append(d...)
		vals[i] = obj
	}
	list, d := types.ListValue(elem, vals)
	diags.Append(d...)
	return list, diags
}

func sectionsFromCatalog(ctx context.Context, sections []api_client.ComplianceCatalogSection, remainingDepth int) (types.List, diag.Diagnostics) {
	elem := sectionObjectType(remainingDepth)
	if len(sections) == 0 {
		return types.ListNull(elem), nil
	}
	vals := make([]attr.Value, len(sections))
	var diags diag.Diagnostics
	for i, s := range sections {
		tests, d := testsFromCatalog(ctx, s.Tests)
		diags.Append(d...)
		attrs := map[string]attr.Value{
			"name":  types.StringValue(s.Name),
			"tests": tests,
		}
		if remainingDepth > 0 {
			nested, d := sectionsFromCatalog(ctx, s.Sections, remainingDepth-1)
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

func validateSections(resp *resource.ValidateConfigResponse, list types.List, parent path.Path) {
	if list.IsNull() || list.IsUnknown() {
		return
	}
	for i, e := range list.Elements() {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		p := parent.AtListIndex(i)
		attrs := obj.Attributes()
		if listLen(attrs["tests"]) > 0 && listLen(attrs["sections"]) > 0 {
			resp.Diagnostics.AddAttributeError(p, invalidSectionSummary, mixedSectionMessage)
		}
		if nested, ok := attrs["sections"]; ok {
			if l, ok := asList(nested); ok {
				validateSections(resp, l, p.AtName("sections"))
			}
		}
	}
}

func catalogMissingDiag(id string) diag.Diagnostics {
	var d diag.Diagnostics
	d.AddError(
		"Error reading custom compliance framework catalog",
		fmt.Sprintf("GET /api/compliance/catalog/%s returned no framework; sections cannot be refreshed.", id),
	)
	return d
}
