package custom_compliance_framework

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

// Controls that a custom alert (or the UI) links into this framework arrive as
// extra tests in the catalog tree. This resource does not declare them, so it
// must neither report them as drift nor delete them: PUT replaces the whole
// section tree, and dropping a control there also clears the link on the alert.
//
// Ownership is decided against prior state: whatever this resource last wrote is
// owned, everything else in the catalog is foreign. Sections are matched by name
// and tests by rule id, which is how the API addresses them.

type ownedSection struct {
	rules    map[string]bool
	children ownedIndex
}

type ownedIndex map[string]ownedSection

func indexOwned(sections []api_client.CustomComplianceFrameworkSection) ownedIndex {
	if len(sections) == 0 {
		return nil
	}
	index := make(ownedIndex, len(sections))
	for _, section := range sections {
		rules := make(map[string]bool, len(section.Tests))
		for _, test := range section.Tests {
			rules[test.RuleID] = true
		}
		index[section.Name] = ownedSection{
			rules:    rules,
			children: indexOwned(section.Sections),
		}
	}
	return index
}

// retainOwnedCatalog drops every catalog test and section this resource does not
// own, so that Read reports only the controls the config is responsible for.
func retainOwnedCatalog(remote []api_client.ComplianceCatalogSection, owned ownedIndex) []api_client.ComplianceCatalogSection {
	if len(remote) == 0 {
		return remote
	}
	kept := make([]api_client.ComplianceCatalogSection, 0, len(remote))
	for _, section := range remote {
		match, ok := owned[section.Name]
		if !ok {
			continue
		}
		section.Tests = retainOwnedTests(section.Tests, match.rules)
		section.Sections = retainOwnedCatalog(section.Sections, match.children)
		section.TotalTests = len(section.Tests)
		kept = append(kept, section)
	}
	return kept
}

func retainOwnedTests(tests []api_client.ComplianceCatalogTest, rules map[string]bool) []api_client.ComplianceCatalogTest {
	if len(tests) == 0 {
		return tests
	}
	kept := make([]api_client.ComplianceCatalogTest, 0, len(tests))
	for _, test := range tests {
		if rules[test.RuleID] {
			kept = append(kept, test)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

// foreignSections returns the part of the catalog this resource does not own, in
// request form, so that a write can send it back untouched.
func foreignSections(remote []api_client.ComplianceCatalogSection, owned ownedIndex) []api_client.CustomComplianceFrameworkSection {
	var out []api_client.CustomComplianceFrameworkSection
	for _, section := range remote {
		match, isOwned := owned[section.Name]
		var tests []api_client.CustomComplianceFrameworkTest
		for _, test := range section.Tests {
			if isOwned && match.rules[test.RuleID] {
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
		nested := foreignSections(section.Sections, match.children)
		if len(tests) == 0 && len(nested) == 0 {
			continue
		}
		out = append(out, api_client.CustomComplianceFrameworkSection{
			Name:     section.Name,
			Tests:    tests,
			Sections: nested,
		})
	}
	return out
}

// mergeForeignSections folds foreign controls back into the sections a write is
// about to send, so the request describes the framework as a whole.
func mergeForeignSections(planned, foreign []api_client.CustomComplianceFrameworkSection) []api_client.CustomComplianceFrameworkSection {
	if len(foreign) == 0 {
		return planned
	}
	out := make([]api_client.CustomComplianceFrameworkSection, len(planned))
	copy(out, planned)

	for _, extra := range foreign {
		index := -1
		for i, section := range out {
			if section.Name == extra.Name {
				index = i
				break
			}
		}
		if index < 0 {
			out = append(out, extra)
			continue
		}
		out[index].Tests = append(out[index].Tests, extra.Tests...)
		out[index].Sections = mergeForeignSections(out[index].Sections, extra.Sections)
	}
	return out
}
