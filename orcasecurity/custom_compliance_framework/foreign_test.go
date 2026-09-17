package custom_compliance_framework

import (
	"reflect"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// declared is what this resource wrote: section 1 holding one control.
func declared() ownership {
	return ownership{"1": {Rules: []string{"rc7bcf3b77f"}}}
}

// remote is what the catalog returns: the declared control, one an alert linked
// into the same section, and a section that exists only because of another alert.
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
	kept := retainOwnedCatalog(remote(), declared())

	if len(kept) != 1 || kept[0].ID != "1" {
		t.Fatalf("a section this resource does not own must not reach state: %+v", kept)
	}
	if len(kept[0].Tests) != 1 || kept[0].Tests[0].RuleID != "rc7bcf3b77f" {
		t.Fatalf("only the owned control may reach state: %+v", kept[0].Tests)
	}
	if kept[0].TotalTests != 1 {
		t.Errorf("total_tests must follow the filtered list, got %d", kept[0].TotalTests)
	}
}

func TestForeignSectionsReturnsOnlyWhatOthersOwn(t *testing.T) {
	foreign := foreignSections(remote(), declared())

	if len(foreign) != 2 {
		t.Fatalf("expected the linked control and the alert-only section, got %+v", foreign)
	}
	if foreign[0].ID != "1" || len(foreign[0].Tests) != 1 || foreign[0].Tests[0].RuleID != "ur1111111111" {
		t.Fatalf("the owned control must not be repeated as foreign: %+v", foreign[0])
	}
	if got := foreign[0].Tests[0]; got.RuleIDInFramework != "1.2" || got.Priority != "High" || got.ControlUniqueID != "42" {
		t.Errorf("a foreign control must be sent back as the API returned it: %+v", got)
	}
	if foreign[1].ID != "2" {
		t.Fatalf("expected the alert-only section, got %+v", foreign[1])
	}
}

func TestMergeForeignSectionsFoldsControlsBackIntoTheRequest(t *testing.T) {
	request := []api_client.CustomComplianceFrameworkSection{{
		Name:  "section_1",
		Tests: []api_client.CustomComplianceFrameworkTest{{RuleID: "rc7bcf3b77f", RuleIDInFramework: "1.1"}},
	}}
	merged := mergeForeignSections(request, []identifiedSection{{ID: "1"}}, foreignSections(remote(), declared()))

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
	request := []api_client.CustomComplianceFrameworkSection{{
		Name:  "section_1",
		Tests: []api_client.CustomComplianceFrameworkTest{{RuleID: "keeper", RuleIDInFramework: "1.1"}},
	}}
	merged := mergeForeignSections(request, []identifiedSection{{ID: "1"}}, foreignSections(remote(), declared()))

	for _, test := range merged[0].Tests {
		if test.RuleID == "rc7bcf3b77f" {
			t.Fatal("a control dropped from the config must stay dropped")
		}
	}
}

// The API accepts sibling sections that share a display name and tells them
// apart by id, so ownership has to as well.
func TestOwnershipKeepsDuplicateSiblingNamesApart(t *testing.T) {
	catalog := []api_client.ComplianceCatalogSection{
		{ID: "1", Name: "dup", Tests: []api_client.ComplianceCatalogTest{{RuleID: "mine", ReferenceID: "1.1"}}},
		{ID: "2", Name: "dup", Tests: []api_client.ComplianceCatalogTest{{RuleID: "theirs", ReferenceID: "2.1"}}},
	}
	owned := ownership{"1": {Rules: []string{"mine"}}}

	kept := retainOwnedCatalog(catalog, owned)
	if len(kept) != 1 || kept[0].ID != "1" || len(kept[0].Tests) != 1 || kept[0].Tests[0].RuleID != "mine" {
		t.Fatalf("the owned section must survive intact, not be matched by name: %+v", kept)
	}

	foreign := foreignSections(catalog, owned)
	if len(foreign) != 1 || foreign[0].ID != "2" || foreign[0].Tests[0].RuleID != "theirs" {
		t.Fatalf("only the second section is foreign: %+v", foreign)
	}

	request := []api_client.CustomComplianceFrameworkSection{{
		Name:  "dup",
		Tests: []api_client.CustomComplianceFrameworkTest{{RuleID: "mine", RuleIDInFramework: "1.1"}},
	}}
	merged := mergeForeignSections(request, []identifiedSection{{ID: "1"}}, foreign)
	if len(merged) != 2 {
		t.Fatalf("the foreign section must be added, not merged into its namesake: %+v", merged)
	}
	if len(merged[0].Tests) != 1 {
		t.Fatalf("the owned section must keep exactly its own control: %+v", merged[0])
	}
}

func TestOwnershipFromSectionsResolvesIDs(t *testing.T) {
	owned := ownershipFromSections(oneSection(t))
	section, ok := owned["1"]
	if !ok {
		t.Fatalf("expected the first section to resolve to id 1, got %v", owned)
	}
	if !section.owns("r1") {
		t.Fatalf("expected r1 to be owned, got %+v", section)
	}
	if ids := identifySections(oneSection(t)); len(ids) != 1 || ids[0].ID != "1" {
		t.Fatalf("request ids must match: %+v", ids)
	}
}

func TestOwnershipFromNestedSections(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	childType := sectionObjectType(maxSectionDepth - 1)
	child := mustObject(t, childType, map[string]attr.Value{
		"name":                    types.StringValue("child"),
		"section_id_in_framework": types.StringNull(),
		"tests":                   mustList(t, testObjectType(), testObj(t, "nested", "1.1.1")),
		"sections":                types.ListNull(sectionObjectType(maxSectionDepth - 2)),
	})
	list := mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
		"name":                    types.StringValue("parent"),
		"section_id_in_framework": types.StringNull(),
		"tests":                   types.ListNull(testObjectType()),
		"sections":                mustList(t, childType, child),
	}))

	owned := ownershipFromSections(list)
	if !owned["1"].Sections["1.1"].owns("nested") {
		t.Fatalf("nested ownership must resolve to 1.1: %+v", owned)
	}
}
