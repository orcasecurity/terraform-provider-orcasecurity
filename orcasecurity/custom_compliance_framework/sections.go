package custom_compliance_framework

import (
	"fmt"
	"strconv"
	"strings"

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
// sub_sub_category). schemaSectionDepth includes one extra nested `sections`
// attribute so a fourth level is visible to Terraform and rejected instead
// of being silently discarded. Catalog data sources use maxCatalogDepth the
// other way (maxCatalogDepth-1) because they have no rejected extra level.
const maxSectionDepth = 3
const schemaSectionDepth = maxSectionDepth + 1

const mixedSectionMessage = "a section cannot have both tests and sub-sections; the API would silently flatten it and the sub-section would inherit the parent name"

const leafNeedsTestMessage = "every leaf section must have at least one test; the API drops a section with no controls, so Terraform would see it disappear"

const emptyTestsListMessage = "an empty tests list is read back as null; omit the attribute instead of setting tests = []"

const emptyNestedSectionsMessage = "an empty nested sections list is read back as null; omit the attribute instead of setting sections = []"

const emptyRootSectionsMessage = "a framework must contain at least one section"

const fourthLevelMessage = "the API stores three levels; move these controls up one level"

const invalidSectionIDMessage = "section_id_in_framework must be an integer, and a nested section must extend its parent (e.g. 7.2); the API derives section ids from control ids and rejects a non-numeric part"

const duplicateSectionIDMessage = "section_id_in_framework must be unique among siblings; the API merges sections that share an id, so Terraform would see a smaller tree"

const nestedSectionsDescription = "Nested sub-sections. A section may have tests or sub-sections, never both."

const rejectedLevelDescription = "Not supported — the API stores three levels; ValidateConfig rejects controls placed here."

const invalidSectionSummary = "Invalid section"

func nestedSectionsAttrDescription(remainingDepth int) string {
	if remainingDepth == 1 {
		return rejectedLevelDescription
	}
	return nestedSectionsDescription
}

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

func sectionDepth(parentID string) int {
	if parentID == "" {
		return 1
	}
	return strings.Count(parentID, ".") + 2
}

func isPlainUint(s string) bool {
	_, err := strconv.ParseUint(s, 10, 64)
	return err == nil
}

// sectionIDMatchesDepth is the server's rule: one unsigned integer per level,
// and each nested id must extend its parent. Empty id is omitted (positional).
func sectionIDMatchesDepth(id, parentID string, depth int) bool {
	if id == "" {
		return true
	}
	if depth <= 1 {
		return isPlainUint(id)
	}
	prefix := parentID + "."
	if parentID == "" || !strings.HasPrefix(id, prefix) {
		return false
	}
	return isPlainUint(id[len(prefix):])
}

type resolvedSectionID struct {
	ID    string
	Valid bool
}

// resolveSiblingIDs assigns ids for one sibling list. Explicit values win;
// omitted values take the next unused positional id so they cannot collide
// with an earlier explicit sibling (B8 shape 2).
func resolveSiblingIDs(elems []attr.Value, parentID string, depth int) []resolvedSectionID {
	out := make([]resolvedSectionID, len(elems))
	userIDs := make([]string, len(elems))
	claimed := make(map[string]struct{}, len(elems))

	for i, e := range elems {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			out[i].Valid = true
			continue
		}
		userIDs[i] = attrString(obj.Attributes(), "section_id_in_framework")
		if userIDs[i] == "" {
			continue
		}
		if !sectionIDMatchesDepth(userIDs[i], parentID, depth) {
			out[i] = resolvedSectionID{ID: positionalSectionID(parentID, i), Valid: false}
			continue
		}
		out[i] = resolvedSectionID{ID: userIDs[i], Valid: true}
		claimed[userIDs[i]] = struct{}{}
	}

	next := 1
	for i := range elems {
		if userIDs[i] != "" {
			continue
		}
		obj, ok := elems[i].(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		for {
			id := positionalSectionID(parentID, next-1)
			next++
			if _, taken := claimed[id]; taken {
				continue
			}
			out[i] = resolvedSectionID{ID: id, Valid: true}
			claimed[id] = struct{}{}
			break
		}
	}
	return out
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
	elems := list.Elements()
	ids := resolveSiblingIDs(elems, parentID, sectionDepth(parentID))
	out := make([]api_client.CustomComplianceFrameworkSection, 0, len(elems))
	for i, e := range elems {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		attrs := obj.Attributes()
		id := ids[i].ID
		if id == "" {
			id = positionalSectionID(parentID, i)
		}
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

func testsFromCatalog(tests []api_client.ComplianceCatalogTest) (types.List, diag.Diagnostics) {
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

func sectionsFromCatalog(sections []api_client.ComplianceCatalogSection, remainingDepth int) (types.List, diag.Diagnostics) {
	elem := sectionObjectType(remainingDepth)
	if len(sections) == 0 {
		return types.ListNull(elem), nil
	}
	vals := make([]attr.Value, len(sections))
	var diags diag.Diagnostics
	for i, s := range sections {
		if remainingDepth == 0 {
			obj, d := types.ObjectValue(elem.AttrTypes, map[string]attr.Value{
				"name": types.StringValue(s.Name),
			})
			diags.Append(d...)
			vals[i] = obj
			continue
		}
		tests, d := testsFromCatalog(s.Tests)
		diags.Append(d...)
		attrs := map[string]attr.Value{
			"name":                    types.StringValue(s.Name),
			"section_id_in_framework": tfconv.StringOrNull(s.ID),
			"tests":                   tests,
		}
		if remainingDepth > 0 {
			nested, d := sectionsFromCatalog(s.Sections, remainingDepth-1)
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

func listKnownEmpty(v attr.Value) bool {
	l, ok := v.(types.List)
	if !ok || l.IsNull() || l.IsUnknown() {
		return false
	}
	return len(l.Elements()) == 0
}

func listUnknown(v attr.Value) bool {
	return v != nil && v.IsUnknown()
}

func validateSections(resp *resource.ValidateConfigResponse, list types.List, parent path.Path) {
	validateSectionsAt(resp, list, parent, 1, "")
}

func validateSectionsAt(resp *resource.ValidateConfigResponse, list types.List, parent path.Path, depth int, parentID string) {
	if list.IsNull() || list.IsUnknown() {
		return
	}
	if len(list.Elements()) == 0 {
		msg := emptyRootSectionsMessage
		if depth > 1 {
			msg = emptyNestedSectionsMessage
		}
		resp.Diagnostics.AddAttributeError(parent, invalidSectionSummary, msg)
		return
	}
	ids := resolveSiblingIDs(list.Elements(), parentID, depth)
	reportSiblingIDErrors(resp, list.Elements(), parent, ids)
	for i, e := range list.Elements() {
		validateOneSection(resp, e, parent.AtListIndex(i), depth, ids[i].ID)
	}
}

func reportSiblingIDErrors(resp *resource.ValidateConfigResponse, elems []attr.Value, parent path.Path, ids []resolvedSectionID) {
	seen := make(map[string]int, len(ids))
	for i, r := range ids {
		obj, ok := elems[i].(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		p := parent.AtListIndex(i).AtName("section_id_in_framework")
		userID := attrString(obj.Attributes(), "section_id_in_framework")
		if userID != "" && !r.Valid {
			resp.Diagnostics.AddAttributeError(p, invalidSectionSummary, invalidSectionIDMessage)
			continue
		}
		if r.ID == "" || !r.Valid {
			continue
		}
		if _, dup := seen[r.ID]; dup {
			resp.Diagnostics.AddAttributeError(p, invalidSectionSummary, duplicateSectionIDMessage)
			continue
		}
		seen[r.ID] = i
	}
}

func validateOneSection(resp *resource.ValidateConfigResponse, e attr.Value, p path.Path, depth int, resolved string) {
	obj, ok := e.(types.Object)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return
	}
	if depth > maxSectionDepth {
		resp.Diagnostics.AddAttributeError(p, invalidSectionSummary, fourthLevelMessage)
		return
	}
	attrs := obj.Attributes()
	if listKnownEmpty(attrs["tests"]) {
		resp.Diagnostics.AddAttributeError(p.AtName("tests"), invalidSectionSummary, emptyTestsListMessage)
		if listLen(attrs["sections"]) == 0 {
			return
		}
	}
	if listKnownEmpty(attrs["sections"]) {
		resp.Diagnostics.AddAttributeError(p.AtName("sections"), invalidSectionSummary, emptyNestedSectionsMessage)
		return
	}
	// Data-source for-expressions are unknown during ValidateConfig.
	if listUnknown(attrs["tests"]) || listUnknown(attrs["sections"]) {
		validateNestedSections(resp, attrs, p, depth, resolved)
		return
	}
	nTests, nSections := listLen(attrs["tests"]), listLen(attrs["sections"])
	if nTests > 0 && nSections > 0 {
		resp.Diagnostics.AddAttributeError(p, invalidSectionSummary, mixedSectionMessage)
		return
	}
	if nSections > 0 {
		validateNestedSections(resp, attrs, p, depth, resolved)
		return
	}
	if nTests == 0 {
		resp.Diagnostics.AddAttributeError(p, invalidSectionSummary, leafNeedsTestMessage)
	}
}

func validateNestedSections(resp *resource.ValidateConfigResponse, attrs map[string]attr.Value, p path.Path, depth int, parentID string) {
	nested, ok := asList(attrs["sections"])
	if ok {
		validateSectionsAt(resp, nested, p.AtName("sections"), depth+1, parentID)
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
