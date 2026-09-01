package custom_compliance_framework

import (
	"encoding/json"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

func objectString(obj types.Object, name string) string {
	v, ok := obj.Attributes()[name]
	if !ok {
		return ""
	}
	s, ok := v.(types.String)
	if !ok {
		return ""
	}
	return s.ValueString()
}

func objectList(obj types.Object, name string) types.List {
	v, ok := obj.Attributes()[name]
	if !ok {
		return types.ListNull(types.StringType)
	}
	l, ok := v.(types.List)
	if !ok {
		return types.ListNull(types.StringType)
	}
	return l
}

func TestSectionsRoundTripThreeLevels(t *testing.T) {
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
	got, d := sectionsFromCatalog(catalog, maxSectionDepth-1)
	if d.HasError() {
		t.Fatal(d)
	}
	if len(got.Elements()) != 2 {
		t.Fatalf("got %d sections", len(got.Elements()))
	}
	parent := got.Elements()[0].(types.Object)
	if objectString(parent, "name") != "Parent" {
		t.Fatalf("got %+v", parent.Attributes())
	}
	if !objectList(parent, "tests").IsNull() {
		t.Errorf("parent tests must be null, got %#v", objectList(parent, "tests"))
	}
	flat := got.Elements()[1].(types.Object)
	if !objectList(flat, "sections").IsNull() {
		t.Errorf("leaf nested sections must be null, got %#v", objectList(flat, "sections"))
	}
	child0 := objectList(parent, "sections").Elements()[0].(types.Object)
	test0 := objectList(child0, "tests").Elements()[0].(types.Object)
	if objectString(test0, "rule_id_in_framework") != "1.1.1" {
		t.Errorf("reference_id must map to rule_id_in_framework, got %s", objectString(test0, "rule_id_in_framework"))
	}
	if !test0.Attributes()["priority"].IsNull() {
		t.Error("omitted catalog test fields must be null, not empty string")
	}

	req := sectionsToAPI(got)
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
}

func TestTestsFromCatalog_UIShapedIdentifier(t *testing.T) {
	got, d := testsFromCatalog([]api_client.ComplianceCatalogTest{{
		RuleID: "r1", ReferenceID: "CC6.1",
	}})
	if d.HasError() {
		t.Fatal(d)
	}
	obj := got.Elements()[0].(types.Object)
	if objectString(obj, "rule_id_in_framework") != "CC6.1" {
		t.Fatalf("catalog reference_id must land on rule_id_in_framework, got %s", objectString(obj, "rule_id_in_framework"))
	}
	raw, err := json.Marshal(testsToAPI("1", got))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"reference_id"`) {
		t.Errorf("write payload must not emit reference_id, got %s", raw)
	}
}

func TestSectionsToAPIDerivesRuleIDInFramework(t *testing.T) {
	typ := testObjectType()
	nulls := map[string]attr.Value{
		"priority":            types.StringNull(),
		"control_unique_id":   types.StringNull(),
		"origin_framework_id": types.StringNull(),
	}
	test1 := mustObject(t, typ, map[string]attr.Value{
		"rule_id":              types.StringValue("r1"),
		"rule_id_in_framework": types.StringNull(),
		"priority":             nulls["priority"],
		"control_unique_id":    nulls["control_unique_id"],
		"origin_framework_id":  nulls["origin_framework_id"],
	})
	test2 := mustObject(t, typ, map[string]attr.Value{
		"rule_id":              types.StringValue("r2"),
		"rule_id_in_framework": types.StringNull(),
		"priority":             nulls["priority"],
		"control_unique_id":    nulls["control_unique_id"],
		"origin_framework_id":  nulls["origin_framework_id"],
	})
	rootType := sectionObjectType(maxSectionDepth - 1)
	plan := mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
		"name":     types.StringValue("Flat"),
		"tests":    mustList(t, typ, test1, test2),
		"sections": types.ListNull(sectionObjectType(maxSectionDepth - 2)),
	}))
	got := sectionsToAPI(plan)
	if got[0].Tests[0].RuleIDInFramework != "1.1" || got[0].Tests[1].RuleIDInFramework != "1.2" {
		t.Errorf("derived ids: %+v", got[0].Tests)
	}
}

func TestSectionsFromCatalog_EmptyListsAreNull(t *testing.T) {
	got, d := sectionsFromCatalog([]api_client.ComplianceCatalogSection{{
		Name: "Empty", Tests: []api_client.ComplianceCatalogTest{},
	}}, maxSectionDepth-1)
	if d.HasError() {
		t.Fatal(d)
	}
	parent := got.Elements()[0].(types.Object)
	if !objectList(parent, "tests").IsNull() {
		t.Errorf("empty catalog tests must be null (the B1 mismatch), got %#v", objectList(parent, "tests"))
	}
}

func TestSectionsFromCatalog_MapsSectionID(t *testing.T) {
	got, d := sectionsFromCatalog([]api_client.ComplianceCatalogSection{{
		ID: "7", Name: "Alpha", Tests: []api_client.ComplianceCatalogTest{{RuleID: "r1", ReferenceID: "7.1"}},
	}}, schemaSectionDepth-1)
	if d.HasError() {
		t.Fatal(d)
	}
	sec := got.Elements()[0].(types.Object)
	if objectString(sec, "section_id_in_framework") != "7" {
		t.Errorf("catalog id must map to section_id_in_framework, got %q", objectString(sec, "section_id_in_framework"))
	}
	req := sectionsToAPI(got)
	if req[0].Tests[0].RuleIDInFramework != "7.1" {
		t.Errorf("catalog section id must prefix derived controls on write-back, got %+v", req[0].Tests)
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "section_id_in_framework") {
		t.Errorf("write payload must not send the ignored key, got %s", raw)
	}
}
