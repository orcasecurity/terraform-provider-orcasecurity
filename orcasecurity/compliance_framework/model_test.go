package compliance_framework

import (
	"context"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ptr[T any](v T) *T { return &v }

func TestFilterAndSort(t *testing.T) {
	ctx := context.Background()
	all := map[string]api_client.ComplianceFramework{
		"b": {ID: "b", DisplayName: "Beta", Custom: true, Active: true, Description: ptr("beta desc"), SelectionScopes: []string{"user"}},
		"a": {ID: "a", DisplayName: "Alpha", Custom: false, Active: false, Type: ptr("Orca Frameworks"), SelectionScopes: []string{}},
		"c": {ID: "c", DisplayName: "Gamma", Custom: true, Active: false, Description: ptr("other"), SelectionScopes: []string{}},
		"d": {ID: "d", DisplayName: "PCI DSS 4.0.0", VersionAgnosticDisplayName: ptr("PCI DSS"), Custom: false, Active: false, SelectionScopes: []string{}},
	}

	got, d := filterAndSort(ctx, all, frameworkFilters{})
	if d.HasError() {
		t.Fatal(d)
	}
	if len(got) != 4 || got[0].ID.ValueString() != "a" || got[3].ID.ValueString() != "d" {
		t.Fatalf("sorted ids: %v %v %v %v", got[0].ID, got[1].ID, got[2].ID, got[3].ID)
	}

	custom, d := filterAndSort(ctx, all, frameworkFilters{custom: types.BoolValue(true), active: types.BoolValue(true)})
	if d.HasError() || len(custom) != 1 || custom[0].ID.ValueString() != "b" {
		t.Fatalf("custom+active: %+v %v", custom, d)
	}

	byType, d := filterAndSort(ctx, all, frameworkFilters{typ: types.StringValue("Orca Frameworks")})
	if d.HasError() || len(byType) != 1 || byType[0].ID.ValueString() != "a" {
		t.Fatalf("type filter: %+v %v", byType, d)
	}

	search, d := filterAndSort(ctx, all, frameworkFilters{search: types.StringValue("BETA")})
	if d.HasError() || len(search) != 1 || search[0].ID.ValueString() != "b" {
		t.Fatalf("search: %+v %v", search, d)
	}

	family, d := filterAndSort(ctx, all, frameworkFilters{versionAgnosticDisplayName: types.StringValue("PCI DSS")})
	if d.HasError() || len(family) != 1 || family[0].ID.ValueString() != "d" {
		t.Fatalf("version_agnostic_display_name: %+v %v", family, d)
	}
}

func TestFrameworkToModel_NullOptionalFields(t *testing.T) {
	m, d := frameworkToModel(context.Background(), api_client.ComplianceFramework{
		ID: "minimal", DisplayName: "Minimal", Custom: true, Active: false, SelectionScopes: []string{},
	})
	if d.HasError() {
		t.Fatal(d)
	}
	if !m.Type.IsNull() || !m.Version.IsNull() || !m.IsReady.IsNull() || !m.Visibility.IsNull() {
		t.Errorf("optional fields must be null, got %+v", m)
	}
	if !m.OriginType.IsNull() || !m.CreatedAt.IsNull() || !m.IsForcedCloudVendors.IsNull() {
		t.Errorf("audit/enforcement fields must be null, got %+v", m)
	}
	if m.Description.ValueString() != "" && !m.Description.IsNull() {
		t.Errorf("missing description must be null, got %v", m.Description)
	}
}

func TestFrameworkToModel_DerivesActiveFromScopes(t *testing.T) {
	staleTrue, d := frameworkToModel(context.Background(), api_client.ComplianceFramework{
		ID: "stale", DisplayName: "Stale", Active: true, SelectionScopes: []string{},
	})
	if d.HasError() {
		t.Fatal(d)
	}
	if staleTrue.Active.ValueBool() {
		t.Error("empty selection_scopes must be inactive even when the API flag is true")
	}

	staleFalse, d := frameworkToModel(context.Background(), api_client.ComplianceFramework{
		ID: "held", DisplayName: "Held", Active: false, SelectionScopes: []string{"user"},
	})
	if d.HasError() {
		t.Fatal(d)
	}
	if !staleFalse.Active.ValueBool() {
		t.Error("non-empty selection_scopes must be active even when the API flag is false")
	}

	ctx := context.Background()
	all := map[string]api_client.ComplianceFramework{
		"stale": {ID: "stale", DisplayName: "Stale", Active: true, SelectionScopes: []string{}},
		"held":  {ID: "held", DisplayName: "Held", Active: false, SelectionScopes: []string{"organization"}},
	}
	inactive, d := filterAndSort(ctx, all, frameworkFilters{active: types.BoolValue(false)})
	if d.HasError() || len(inactive) != 1 || inactive[0].ID.ValueString() != "stale" {
		t.Fatalf("active=false must follow scopes, got %+v %v", inactive, d)
	}
	active, d := filterAndSort(ctx, all, frameworkFilters{active: types.BoolValue(true)})
	if d.HasError() || len(active) != 1 || active[0].ID.ValueString() != "held" {
		t.Fatalf("active=true must follow scopes, got %+v %v", active, d)
	}
}

func TestCatalogSectionsLeafAbsent(t *testing.T) {
	got, d := catalogSectionsToModel(context.Background(), []api_client.ComplianceCatalogSection{{
		Name:  "Flat",
		Tests: []api_client.ComplianceCatalogTest{{RuleID: "r1", ReferenceID: "1.1"}},
	}}, maxCatalogDepth-1)
	if d.HasError() {
		t.Fatal(d)
	}
	sec := got.Elements()[0].(types.Object)
	if !sec.Attributes()["sections"].IsNull() {
		t.Errorf("leaf sections must be null, got %#v", sec.Attributes()["sections"])
	}
	test0 := sec.Attributes()["tests"].(types.List).Elements()[0].(types.Object)
	if test0.Attributes()["reference_id"].(types.String).ValueString() != "1.1" {
		t.Errorf("reference_id: %s", test0.Attributes()["reference_id"])
	}
	if !test0.Attributes()["cis_level"].IsNull() {
		t.Errorf("omitted cis_level must be null, got %v", test0.Attributes()["cis_level"])
	}
	if !sec.Attributes()["id"].IsNull() {
		t.Errorf("omitted section id must be null, got %v", sec.Attributes()["id"])
	}
}

func TestCatalogTestsToModel_CISLevel(t *testing.T) {
	got, d := catalogTestsToModel(context.Background(), []api_client.ComplianceCatalogTest{{
		RuleID: "r1", CISLevel: []string{"Level 1"},
	}})
	if d.HasError() {
		t.Fatal(d)
	}
	obj := got.Elements()[0].(types.Object)
	levels := obj.Attributes()["cis_level"].(types.List)
	if levels.IsNull() || len(levels.Elements()) != 1 || levels.Elements()[0].(types.String).ValueString() != "Level 1" {
		t.Errorf("cis_level: %#v", obj.Attributes()["cis_level"])
	}
}
