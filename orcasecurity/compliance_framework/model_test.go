package compliance_framework

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ptr[T any](v T) *T { return &v }

func TestFilterAndSort(t *testing.T) {
	all := map[string]api_client.ComplianceFramework{
		"b": {ID: "b", DisplayName: "Beta", Custom: true, Active: true, Description: ptr("beta desc"), SelectionScopes: []string{"user"}},
		"a": {ID: "a", DisplayName: "Alpha", Custom: false, Active: false, Type: ptr("Orca Frameworks"), SelectionScopes: []string{}},
		"c": {ID: "c", DisplayName: "Gamma", Custom: true, Active: false, Description: ptr("other"), SelectionScopes: []string{}},
	}

	got := filterAndSort(all, frameworkFilters{})
	if len(got) != 3 || got[0].ID.ValueString() != "a" || got[2].ID.ValueString() != "c" {
		t.Fatalf("sorted ids: %v %v %v", got[0].ID, got[1].ID, got[2].ID)
	}

	custom := filterAndSort(all, frameworkFilters{custom: types.BoolValue(true), active: types.BoolValue(true)})
	if len(custom) != 1 || custom[0].ID.ValueString() != "b" {
		t.Fatalf("custom+active: %+v", custom)
	}

	byType := filterAndSort(all, frameworkFilters{typ: types.StringValue("Orca Frameworks")})
	if len(byType) != 1 || byType[0].ID.ValueString() != "a" {
		t.Fatalf("type filter: %+v", byType)
	}

	search := filterAndSort(all, frameworkFilters{search: types.StringValue("BETA")})
	if len(search) != 1 || search[0].ID.ValueString() != "b" {
		t.Fatalf("search: %+v", search)
	}
}

func TestFrameworkToModel_NullOptionalFields(t *testing.T) {
	m := frameworkToModel(api_client.ComplianceFramework{
		ID: "minimal", DisplayName: "Minimal", Custom: true, Active: false, SelectionScopes: []string{},
	})
	if !m.Type.IsNull() || !m.Version.IsNull() || !m.IsReady.IsNull() || !m.Visibility.IsNull() {
		t.Errorf("optional fields must be null, got %+v", m)
	}
	if m.Description.ValueString() != "" && !m.Description.IsNull() {
		t.Errorf("missing description must be null, got %v", m.Description)
	}
}

func TestCatalogSectionsLeafAbsent(t *testing.T) {
	got := catalogSectionsToModel([]api_client.ComplianceCatalogSection{{
		Name:  "Flat",
		Tests: []api_client.ComplianceCatalogTest{{RuleID: "r1", ReferenceID: "1.1"}},
	}})
	if got[0].Sections != nil {
		t.Errorf("leaf sections must be nil, got %#v", got[0].Sections)
	}
	if got[0].Tests[0].ReferenceID.ValueString() != "1.1" {
		t.Errorf("reference_id: %s", got[0].Tests[0].ReferenceID)
	}
}
