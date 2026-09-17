package custom_compliance_framework

import (
	"reflect"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

// declared is what this resource wrote: one section holding one control.
func declared() []api_client.CustomComplianceFrameworkSection {
	return []api_client.CustomComplianceFrameworkSection{{
		Name:  "section_1",
		Tests: []api_client.CustomComplianceFrameworkTest{{RuleID: "rc7bcf3b77f", RuleIDInFramework: "1.1"}},
	}}
}

// remote is what the catalog returns: the declared control plus one an alert
// linked in, and a whole section that only exists because of another alert.
func remote() []api_client.ComplianceCatalogSection {
	return []api_client.ComplianceCatalogSection{
		{
			ID:   "1",
			Name: "section_1",
			Tests: []api_client.ComplianceCatalogTest{
				{RuleID: "rc7bcf3b77f", ReferenceID: "1.1", Priority: "Medium"},
				{RuleID: "ur1111111111", ReferenceID: "1.2", Priority: "High", ControlUniqueID: "42"},
			},
		},
		{
			ID:    "2",
			Name:  "alert only section",
			Tests: []api_client.ComplianceCatalogTest{{RuleID: "ur2222222222", ReferenceID: "2.1"}},
		},
	}
}

func TestRetainOwnedCatalogDropsControlsThisResourceNeverWrote(t *testing.T) {
	kept := retainOwnedCatalog(remote(), indexOwned(declared()))

	if len(kept) != 1 || kept[0].Name != "section_1" {
		t.Fatalf("a section this resource does not declare must not reach state: %+v", kept)
	}
	if len(kept[0].Tests) != 1 || kept[0].Tests[0].RuleID != "rc7bcf3b77f" {
		t.Fatalf("only the declared control may reach state: %+v", kept[0].Tests)
	}
	if kept[0].TotalTests != 1 {
		t.Errorf("total_tests must follow the filtered list, got %d", kept[0].TotalTests)
	}
}

// Import has no prior state, so nothing is filtered: the caller skips the filter
// entirely. Passing an empty index must therefore keep nothing, not everything —
// that is what makes the caller's null check load-bearing.
func TestRetainOwnedCatalogKeepsNothingWithoutOwnership(t *testing.T) {
	if kept := retainOwnedCatalog(remote(), nil); len(kept) != 0 {
		t.Fatalf("expected nothing retained, got %+v", kept)
	}
}

func TestForeignSectionsReturnsOnlyWhatOthersOwn(t *testing.T) {
	foreign := foreignSections(remote(), indexOwned(declared()))

	if len(foreign) != 2 {
		t.Fatalf("expected the linked control and the alert-only section, got %+v", foreign)
	}
	if foreign[0].Name != "section_1" || len(foreign[0].Tests) != 1 ||
		foreign[0].Tests[0].RuleID != "ur1111111111" {
		t.Fatalf("declared control must not be repeated as foreign: %+v", foreign[0])
	}
	if got := foreign[0].Tests[0]; got.RuleIDInFramework != "1.2" || got.Priority != "High" || got.ControlUniqueID != "42" {
		t.Errorf("a foreign control must be sent back as the API returned it: %+v", got)
	}
	if foreign[1].Name != "alert only section" {
		t.Fatalf("expected the alert-only section, got %+v", foreign[1])
	}
}

func TestMergeForeignSectionsFoldsControlsBackIntoTheRequest(t *testing.T) {
	merged := mergeForeignSections(declared(), foreignSections(remote(), indexOwned(declared())))

	if len(merged) != 2 {
		t.Fatalf("expected the declared section plus the alert-only one, got %+v", merged)
	}
	rules := []string{}
	for _, test := range merged[0].Tests {
		rules = append(rules, test.RuleID)
	}
	if !reflect.DeepEqual(rules, []string{"rc7bcf3b77f", "ur1111111111"}) {
		t.Fatalf("declared control first, then the linked one: %v", rules)
	}
	if merged[1].Name != "alert only section" {
		t.Errorf("the alert-only section must survive the write: %+v", merged[1])
	}
}

// A control removed from the config was owned, so it is not foreign and must not
// come back through the merge.
func TestMergeForeignSectionsDoesNotResurrectRemovedControls(t *testing.T) {
	stillOwned := []api_client.CustomComplianceFrameworkSection{{
		Name:  "section_1",
		Tests: []api_client.CustomComplianceFrameworkTest{{RuleID: "keeper", RuleIDInFramework: "1.1"}},
	}}
	foreign := foreignSections(remote(), indexOwned(declared()))
	merged := mergeForeignSections(stillOwned, foreign)

	for _, test := range merged[0].Tests {
		if test.RuleID == "rc7bcf3b77f" {
			t.Fatal("a control dropped from the config must stay dropped")
		}
	}
}

func TestMergeForeignSectionsNestsIntoSubSections(t *testing.T) {
	owned := []api_client.CustomComplianceFrameworkSection{{
		Name: "parent",
		Sections: []api_client.CustomComplianceFrameworkSection{{
			Name:  "child",
			Tests: []api_client.CustomComplianceFrameworkTest{{RuleID: "mine"}},
		}},
	}}
	catalog := []api_client.ComplianceCatalogSection{{
		Name: "parent",
		Sections: []api_client.ComplianceCatalogSection{{
			Name: "child",
			Tests: []api_client.ComplianceCatalogTest{
				{RuleID: "mine"},
				{RuleID: "theirs", ReferenceID: "1.1.2"},
			},
		}},
	}}

	merged := mergeForeignSections(owned, foreignSections(catalog, indexOwned(owned)))
	child := merged[0].Sections[0]
	if len(child.Tests) != 2 || child.Tests[1].RuleID != "theirs" {
		t.Fatalf("a nested foreign control must be folded into its own section: %+v", child.Tests)
	}
}
