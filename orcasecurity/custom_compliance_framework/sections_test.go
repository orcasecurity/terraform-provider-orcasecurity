package custom_compliance_framework

import (
	"encoding/json"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDeriveRuleIDInFramework(t *testing.T) {
	tests := []struct {
		sectionID string
		index     int
		want      string
	}{
		{"1", 0, "1.1"},
		{"1", 1, "1.2"},
		{"1.1", 0, "1.1.1"},
		{"2", 0, "2.1"},
	}
	for _, tt := range tests {
		if got := DeriveRuleIDInFramework(tt.sectionID, tt.index); got != tt.want {
			t.Errorf("DeriveRuleIDInFramework(%q, %d) = %q, want %q", tt.sectionID, tt.index, got, tt.want)
		}
	}
}

func TestSectionsRoundTripThreeLevels(t *testing.T) {
	plan := []sectionModel{
		{
			Name: types.StringValue("Parent"),
			Sections: []midSectionModel{
				{
					Name: types.StringValue("Child One"),
					Tests: []testModel{
						{RuleID: types.StringValue("r1"), RuleIDInFramework: types.StringValue("1.1.1")},
					},
				},
				{
					Name: types.StringValue("Child Two"),
					Tests: []testModel{
						{RuleID: types.StringValue("r2"), RuleIDInFramework: types.StringValue("1.2.1")},
					},
				},
			},
		},
		{
			Name: types.StringValue("Flat"),
			Tests: []testModel{
				{RuleID: types.StringValue("r3"), RuleIDInFramework: types.StringValue("2.1")},
			},
		},
	}

	req := sectionsToAPI(plan)
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []api_client.CustomComplianceFrameworkSection
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded[0].Name != "Parent" || len(decoded[0].Tests) != 0 {
		t.Errorf("parent: %+v", decoded[0])
	}
	if decoded[0].Sections[0].Tests[0].RuleIDInFramework != "1.1.1" {
		t.Errorf("child test: %+v", decoded[0].Sections[0].Tests)
	}

	// Catalog omits `sections` on leaves (zero value after unmarshal of absent key).
	catalog := []api_client.ComplianceCatalogSection{
		{
			Name: "Parent",
			Sections: []api_client.ComplianceCatalogSection{
				{Name: "Child One", Tests: []api_client.ComplianceCatalogTest{{RuleID: "r1", ReferenceID: "1.1.1"}}},
				{Name: "Child Two", Tests: []api_client.ComplianceCatalogTest{{RuleID: "r2", ReferenceID: "1.2.1"}}},
			},
		},
		{
			Name:  "Flat",
			Tests: []api_client.ComplianceCatalogTest{{RuleID: "r3", ReferenceID: "2.1"}},
		},
	}
	got := sectionsFromCatalog(catalog)
	if len(got) != 2 || got[0].Name.ValueString() != "Parent" {
		t.Fatalf("got %+v", got)
	}
	if got[0].Tests != nil {
		t.Errorf("parent tests must be nil, got %#v", got[0].Tests)
	}
	if got[1].Sections != nil {
		t.Errorf("leaf nested sections must be nil, got %#v", got[1].Sections)
	}
	if got[0].Sections[0].Tests[0].RuleIDInFramework.ValueString() != "1.1.1" {
		t.Errorf("reference_id must map to rule_id_in_framework, got %s", got[0].Sections[0].Tests[0].RuleIDInFramework)
	}
	if !got[0].Sections[0].Tests[0].Priority.IsNull() {
		t.Error("omitted catalog test fields must be null, not empty string")
	}
}

func TestSectionsToAPIDerivesRuleIDInFramework(t *testing.T) {
	plan := []sectionModel{{
		Name: types.StringValue("Flat"),
		Tests: []testModel{
			{RuleID: types.StringValue("r1")},
			{RuleID: types.StringValue("r2")},
		},
	}}
	got := sectionsToAPI(plan)
	if got[0].Tests[0].RuleIDInFramework != "1.1" || got[0].Tests[1].RuleIDInFramework != "1.2" {
		t.Errorf("derived ids: %+v", got[0].Tests)
	}
}

func TestMixedSectionDetector(t *testing.T) {
	if !sectionHasTestsAndChildren(1, 1) {
		t.Error("mixed must be rejected")
	}
	if sectionHasTestsAndChildren(1, 0) || sectionHasTestsAndChildren(0, 1) {
		t.Error("tests XOR sections must be allowed")
	}
}
