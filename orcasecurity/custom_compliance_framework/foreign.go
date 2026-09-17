package custom_compliance_framework

import (
	"context"
	"encoding/json"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Controls that a custom alert (or the Orca UI) links into this framework arrive
// as extra tests in the catalog tree. This resource does not declare them, so it
// must neither report them as drift nor delete them: PUT replaces the whole
// section tree, and dropping a control there also clears the link on the alert.
//
// Ownership is what this resource last wrote, recorded in private state. Prior
// state cannot stand in for it: state written by an earlier provider version
// holds the foreign controls too, so an upgrade would read them as owned and
// delete them on the next apply.
//
// Sections are keyed by the id the API derives for them, never by display name —
// the API accepts sibling sections that share a name.

const ownershipKey = "compliance_controls"

type ownedSection struct {
	Rules    []string                `json:"rules,omitempty"`
	Sections map[string]ownedSection `json:"sections,omitempty"`
}

// ownership maps a section id to the controls this resource wrote into it.
type ownership map[string]ownedSection

func (o ownership) section(id string) (ownedSection, bool) {
	section, ok := o[id]
	return section, ok
}

func (s ownedSection) owns(ruleID string) bool {
	for _, rule := range s.Rules {
		if rule == ruleID {
			return true
		}
	}
	return false
}

// identifiedSection carries the id the API derives for a planned section, so a
// request built by sectionsToAPI can be addressed by id rather than by name.
type identifiedSection struct {
	ID       string
	Children []identifiedSection
}

// ownershipFromSections records what a plan or state describes. It resolves ids
// exactly as sectionsToAPI does, so the two views address the same sections.
func ownershipFromSections(list types.List) ownership {
	return ownershipAt(list, "", 1)
}

func ownershipAt(list types.List, parentID string, depth int) ownership {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	elems := list.Elements()
	ids := resolveSiblingIDs(elems, parentID, depth)
	out := make(ownership, len(elems))
	for i, e := range elems {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		attrs := obj.Attributes()
		id := ids[i].ID
		section := ownedSection{}
		if tests, ok := asList(attrs["tests"]); ok {
			for _, test := range testsToAPI(id, tests) {
				section.Rules = append(section.Rules, test.RuleID)
			}
		}
		if nested, ok := asList(attrs["sections"]); ok {
			section.Sections = ownershipAt(nested, id, depth+1)
		}
		out[id] = section
	}
	return out
}

// identifySections labels the request sectionsToAPI produced. Both walk the same
// elements in the same order, so the ids line up with the request slice.
func identifySections(list types.List) []identifiedSection {
	return identifySectionsAt(list, "", 1)
}

func identifySectionsAt(list types.List, parentID string, depth int) []identifiedSection {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	elems := list.Elements()
	ids := resolveSiblingIDs(elems, parentID, depth)
	var out []identifiedSection
	for i, e := range elems {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		id := ids[i].ID
		var children []identifiedSection
		if nested, ok := asList(obj.Attributes()["sections"]); ok {
			children = identifySectionsAt(nested, id, depth+1)
		}
		out = append(out, identifiedSection{ID: id, Children: children})
	}
	return out
}

// retainOwnedCatalog drops every catalog test and section this resource does not
// own, so Read reports only the controls the config is responsible for.
func retainOwnedCatalog(remote []api_client.ComplianceCatalogSection, owned ownership) []api_client.ComplianceCatalogSection {
	if len(remote) == 0 {
		return remote
	}
	kept := make([]api_client.ComplianceCatalogSection, 0, len(remote))
	for _, section := range remote {
		match, ok := owned.section(section.ID)
		if !ok {
			continue
		}
		section.Tests = retainOwnedTests(section.Tests, match)
		section.Sections = retainOwnedCatalog(section.Sections, match.Sections)
		section.TotalTests = len(section.Tests)
		kept = append(kept, section)
	}
	return kept
}

func retainOwnedTests(tests []api_client.ComplianceCatalogTest, owned ownedSection) []api_client.ComplianceCatalogTest {
	kept := make([]api_client.ComplianceCatalogTest, 0, len(tests))
	for _, test := range tests {
		if owned.owns(test.RuleID) {
			kept = append(kept, test)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

// foreignSection is a catalog subtree this resource does not own, in request
// form plus the id needed to merge it back into the right place.
type foreignSection struct {
	ID       string
	Name     string
	Tests    []api_client.CustomComplianceFrameworkTest
	Children []foreignSection
}

// foreignSections returns the part of the catalog this resource does not own, so
// that a write can send it back untouched.
func foreignSections(remote []api_client.ComplianceCatalogSection, owned ownership) []foreignSection {
	var out []foreignSection
	for _, section := range remote {
		match, isOwned := owned.section(section.ID)
		var tests []api_client.CustomComplianceFrameworkTest
		for _, test := range section.Tests {
			if isOwned && match.owns(test.RuleID) {
				continue
			}
			tests = append(tests, api_client.CustomComplianceFrameworkTest{
				RuleID:            test.RuleID,
				RuleIDInFramework: test.ReferenceID,
				Priority:          test.Priority,
				ControlUniqueID:   test.ControlUniqueID,
				OriginFrameworkID: test.OriginFrameworkID,
			})
		}
		children := foreignSections(section.Sections, match.Sections)
		if len(tests) == 0 && len(children) == 0 {
			continue
		}
		out = append(out, foreignSection{ID: section.ID, Name: section.Name, Tests: tests, Children: children})
	}
	return out
}

// mergeForeignSections folds foreign controls back into the sections a write is
// about to send, matching on section id.
func mergeForeignSections(
	request []api_client.CustomComplianceFrameworkSection,
	ids []identifiedSection,
	foreign []foreignSection,
) []api_client.CustomComplianceFrameworkSection {
	if len(foreign) == 0 {
		return request
	}
	out := make([]api_client.CustomComplianceFrameworkSection, len(request))
	copy(out, request)

	for _, extra := range foreign {
		index := -1
		for i := range out {
			if i < len(ids) && ids[i].ID == extra.ID {
				index = i
				break
			}
		}
		if index < 0 {
			out = append(out, requestFromForeign(extra))
			continue
		}
		out[index].Tests = append(out[index].Tests, extra.Tests...)
		var children []identifiedSection
		if index < len(ids) {
			children = ids[index].Children
		}
		out[index].Sections = mergeForeignSections(out[index].Sections, children, extra.Children)
	}
	return out
}

func requestFromForeign(section foreignSection) api_client.CustomComplianceFrameworkSection {
	out := api_client.CustomComplianceFrameworkSection{Name: section.Name, Tests: section.Tests}
	for _, child := range section.Children {
		out.Sections = append(out.Sections, requestFromForeign(child))
	}
	return out
}

// foreignNames lists the controls a write is preserving, for the diagnostic that
// explains why they are neither reported nor removed.
func foreignRuleIDs(sections []foreignSection) []string {
	var out []string
	for _, section := range sections {
		for _, test := range section.Tests {
			out = append(out, test.RuleID)
		}
		out = append(out, foreignRuleIDs(section.Children)...)
	}
	return out
}

// privateState is the slice of resource private state this resource uses. The
// framework's own type lives in an internal package, so it is named by shape.
type privateState interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

// readOwnership returns what this resource recorded on its last write. ok is
// false for state written before the provider tracked ownership, and for a
// freshly imported resource.
func readOwnership(ctx context.Context, private privateState) (ownership, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if private == nil {
		return nil, false, diags
	}
	raw, d := private.GetKey(ctx, ownershipKey)
	diags.Append(d...)
	if diags.HasError() || len(raw) == 0 {
		return nil, false, diags
	}
	var owned ownership
	if err := json.Unmarshal(raw, &owned); err != nil {
		// Unreadable ownership is treated as absent: the conservative path keeps
		// every control rather than deleting one on a parsing accident.
		return nil, false, diags
	}
	return owned, true, diags
}

func writeOwnership(ctx context.Context, private privateState, sections types.List) diag.Diagnostics {
	var diags diag.Diagnostics
	if private == nil {
		return diags
	}
	raw, err := json.Marshal(ownershipFromSections(sections))
	if err != nil {
		diags.AddError(errReadingFramework, "could not record which controls this framework manages: "+err.Error())
		return diags
	}
	diags.Append(private.SetKey(ctx, ownershipKey, raw)...)
	return diags
}
