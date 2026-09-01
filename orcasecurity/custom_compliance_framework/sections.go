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

const duplicateSectionIDMessage = "section_id_in_framework must be unique and strictly ascending among siblings; the API returns sections sorted by id, so a descending or duplicate list would permute after apply"

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

func lastSegment(id string) string {
	if i := strings.LastIndex(id, "."); i >= 0 {
		return id[i+1:]
	}
	return id
}

func lastUint(id string) (uint64, bool) {
	n, err := strconv.ParseUint(lastSegment(id), 10, 64)
	return n, err == nil
}

func nextAfter(parentID, prevID string) string {
	n := uint64(0)
	if prevID != "" {
		v, ok := lastUint(prevID)
		if !ok {
			// resolveSiblingIDs only stores prev from a valid id. If that
			// invariant is broken, start from 0 so the next id is still 1.
			v = 0
		}
		n = v
	}
	s := strconv.FormatUint(n+1, 10)
	if parentID == "" {
		return s
	}
	return parentID + "." + s
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

// resolveSiblingIDs assigns ids for one sibling list in config order.
// Explicit values are kept as written. Omitted values take the next integer
// above the previous sibling so the list is strictly ascending — the API
// returns sections sorted by id, and a Terraform list would permute otherwise.
func resolveSiblingIDs(elems []attr.Value, parentID string, depth int) []resolvedSectionID {
	out := make([]resolvedSectionID, len(elems))
	prev := ""
	for i, e := range elems {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			out[i].Valid = true
			continue
		}
		userID := attrString(obj.Attributes(), "section_id_in_framework")
		if userID != "" {
			if !sectionIDMatchesDepth(userID, parentID, depth) {
				out[i] = resolvedSectionID{ID: userID, Valid: false}
				continue
			}
			out[i] = resolvedSectionID{ID: userID, Valid: true}
			prev = userID
			continue
		}
		id := nextAfter(parentID, prev)
		out[i] = resolvedSectionID{ID: id, Valid: true}
		prev = id
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

var computedTestAttrs = []string{"rule_id_in_framework", "priority", "control_unique_id", "origin_framework_id"}

func listObjectAt(list types.List, i int) (types.Object, bool) {
	if list.IsNull() || list.IsUnknown() || i >= len(list.Elements()) {
		return types.ObjectNull(map[string]attr.Type{}), false
	}
	obj, ok := list.Elements()[i].(types.Object)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return types.ObjectNull(map[string]attr.Type{}), false
	}
	return obj, true
}

func objectAttr(obj types.Object, name string) (attr.Value, bool) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, false
	}
	v, ok := obj.Attributes()[name]
	if !ok || v == nil {
		return nil, false
	}
	return v, true
}

func attrList(obj types.Object, name string, elem attr.Type) types.List {
	v, ok := objectAttr(obj, name)
	if !ok {
		return types.ListNull(elem)
	}
	l, ok := asList(v)
	if !ok {
		return types.ListNull(elem)
	}
	return l
}

func siblingIDChanged(config, state types.List) bool {
	// Walks max(len(config), len(state)) and skips indexes missing on either
	// side. A length change already forces !hasState on the tail, so an id
	// change at a shifted index is still covered.
	n := 0
	if !config.IsNull() && !config.IsUnknown() {
		n = len(config.Elements())
	}
	if !state.IsNull() && !state.IsUnknown() && len(state.Elements()) > n {
		n = len(state.Elements())
	}
	for i := 0; i < n; i++ {
		cobj, okc := listObjectAt(config, i)
		sobj, oks := listObjectAt(state, i)
		if !okc || !oks {
			continue
		}
		cfgID := attrString(cobj.Attributes(), "section_id_in_framework")
		stID := attrString(sobj.Attributes(), "section_id_in_framework")
		if cfgID != "" && stID != "" && cfgID != stID {
			return true
		}
	}
	return false
}

func plannedComputedString(configV, planV, stateV attr.Value, forceUnknown bool) attr.Value {
	if s, ok := configV.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
		return planV
	}
	if forceUnknown || stateV == nil || stateV.IsNull() {
		return types.StringUnknown()
	}
	return planV
}

// rewriteSectionsPlan marks Optional+Computed section/test attributes unknown
// when config omits them and there is no prior state at that index (or any
// sibling id in the list changed, so omitted ids re-derive ascending). That
// is the list-element case UseStateForUnknown cannot cover.
func rewriteSectionsPlan(config, plan, state types.List, remainingDepth int, forceOmitted bool) (types.List, diag.Diagnostics) {
	if plan.IsNull() || plan.IsUnknown() {
		return plan, nil
	}
	typ := sectionObjectType(remainingDepth)
	listChanged := forceOmitted || siblingIDChanged(config, state)
	out := make([]attr.Value, len(plan.Elements()))
	var diags diag.Diagnostics
	for i, e := range plan.Elements() {
		pobj, ok := e.(types.Object)
		if !ok || pobj.IsNull() || pobj.IsUnknown() {
			out[i] = e
			continue
		}
		cobj, _ := listObjectAt(config, i)
		sobj, hasState := listObjectAt(state, i)
		rewritten, d := rewriteSectionPlan(cobj, pobj, sobj, hasState, remainingDepth, listChanged)
		diags.Append(d...)
		out[i] = rewritten
	}
	list, d := types.ListValue(typ, out)
	diags.Append(d...)
	return list, diags
}

func rewriteSectionPlan(cfg, plan, state types.Object, hasState bool, remainingDepth int, listIDChanged bool) (types.Object, diag.Diagnostics) {
	if remainingDepth == 0 {
		return plan, nil
	}
	pattrs := plan.Attributes()
	attrs := make(map[string]attr.Value, len(pattrs))
	for k, v := range pattrs {
		attrs[k] = v
	}
	cfgID := ""
	if !cfg.IsNull() && !cfg.IsUnknown() {
		cfgID = attrString(cfg.Attributes(), "section_id_in_framework")
	}
	stID := ""
	if hasState {
		stID = attrString(state.Attributes(), "section_id_in_framework")
	}
	thisChanged := cfgID != "" && stID != "" && cfgID != stID
	omitted := cfgID == ""
	// When any sibling id changed, unpin omitted ids so the whole list
	// re-derives in ascending order instead of mixing new ids with state.
	forceSectionID := !hasState || (listIDChanged && omitted)
	cfgSec, hasCfgSec := objectAttr(cfg, "section_id_in_framework")
	if !hasCfgSec {
		cfgSec = nil
	}
	stSec, hasStSec := objectAttr(state, "section_id_in_framework")
	if !hasStSec {
		stSec = nil
	}
	attrs["section_id_in_framework"] = plannedComputedString(
		cfgSec,
		pattrs["section_id_in_framework"],
		stSec,
		forceSectionID,
	)
	var diags diag.Diagnostics
	unpinDescendants := thisChanged || (listIDChanged && omitted)
	if tests, ok := asList(pattrs["tests"]); ok {
		rewritten, d := rewriteTestsPlan(attrList(cfg, "tests", testObjectType()), tests, attrList(state, "tests", testObjectType()), unpinDescendants)
		diags.Append(d...)
		attrs["tests"] = rewritten
	}
	if nested, ok := asList(pattrs["sections"]); ok {
		childType := sectionObjectType(remainingDepth - 1)
		rewritten, d := rewriteSectionsPlan(attrList(cfg, "sections", childType), nested, attrList(state, "sections", childType), remainingDepth-1, unpinDescendants)
		diags.Append(d...)
		attrs["sections"] = rewritten
	}
	obj, d := types.ObjectValue(sectionObjectType(remainingDepth).AttrTypes, attrs)
	diags.Append(d...)
	return obj, diags
}

func rewriteTestsPlan(config, plan, state types.List, sectionChanged bool) (types.List, diag.Diagnostics) {
	if plan.IsNull() || plan.IsUnknown() {
		return plan, nil
	}
	elem := testObjectType()
	out := make([]attr.Value, len(plan.Elements()))
	var diags diag.Diagnostics
	for i, e := range plan.Elements() {
		pobj, ok := e.(types.Object)
		if !ok || pobj.IsNull() || pobj.IsUnknown() {
			out[i] = e
			continue
		}
		cobj, _ := listObjectAt(config, i)
		sobj, hasState := listObjectAt(state, i)
		pattrs := pobj.Attributes()
		attrs := make(map[string]attr.Value, len(pattrs))
		for k, v := range pattrs {
			attrs[k] = v
		}
		for _, name := range computedTestAttrs {
			force := !hasState || (sectionChanged && name == "rule_id_in_framework")
			cv, hasCfg := objectAttr(cobj, name)
			if !hasCfg {
				cv = nil
			}
			sv, hasSt := objectAttr(sobj, name)
			if !hasSt {
				sv = nil
			}
			attrs[name] = plannedComputedString(cv, pattrs[name], sv, force)
		}
		obj, d := types.ObjectValue(elem.AttrTypes, attrs)
		diags.Append(d...)
		out[i] = obj
	}
	list, d := types.ListValue(elem, out)
	diags.Append(d...)
	return list, diags
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
	return sectionsToAPIAt(list, "", 1)
}

func sectionsToAPIAt(list types.List, parentID string, depth int) []api_client.CustomComplianceFrameworkSection {
	if list.IsNull() || list.IsUnknown() {
		return []api_client.CustomComplianceFrameworkSection{}
	}
	elems := list.Elements()
	ids := resolveSiblingIDs(elems, parentID, depth)
	out := make([]api_client.CustomComplianceFrameworkSection, 0, len(elems))
	for i, e := range elems {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		attrs := obj.Attributes()
		id := ids[i].ID
		tests := []api_client.CustomComplianceFrameworkTest{}
		if l, ok := asList(attrs["tests"]); ok {
			tests = testsToAPI(id, l)
		}
		nested := []api_client.CustomComplianceFrameworkSection{}
		if l, ok := asList(attrs["sections"]); ok {
			nested = sectionsToAPIAt(l, id, depth+1)
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
	// Order is the last segment only: every resolved sibling already shares
	// the same parent prefix, so the suffix is the numeric sort key.
	var prev uint64
	hasPrev := false
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
		n, ok := lastUint(r.ID)
		if !ok {
			continue
		}
		if hasPrev && n <= prev {
			resp.Diagnostics.AddAttributeError(p, invalidSectionSummary, duplicateSectionIDMessage)
			continue
		}
		prev = n
		hasPrev = true
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
