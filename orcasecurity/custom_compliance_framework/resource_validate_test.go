package custom_compliance_framework

import (
	"context"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateConfig_RejectsMixedTestsAndSections(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	childType := sectionObjectType(maxSectionDepth - 1)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("mixed"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":  types.StringValue("Sec A"),
			"tests": mustList(t, testObjectType(), testObj(t, "r1", "1.1")),
			"sections": mustList(t, childType, mustObject(t, childType, map[string]attr.Value{
				"name":     types.StringValue("Sub A1"),
				"tests":    mustList(t, testObjectType(), testObj(t, "r2", "2.1")),
				"sections": types.ListNull(sectionObjectType(maxSectionDepth - 2)),
			})),
		})),
	})

	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("mixed tests+sections must be a config error")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Detail(), mixedSectionMessage) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected flatten warning, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_AcceptsThreeLevels(t *testing.T) {
	l3Type := sectionObjectType(1)
	l2Type := sectionObjectType(2)
	l1Type := sectionObjectType(3)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("three"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, l1Type, mustObject(t, l1Type, map[string]attr.Value{
			"name":  types.StringValue("L1"),
			"tests": types.ListNull(testObjectType()),
			"sections": mustList(t, l2Type, mustObject(t, l2Type, map[string]attr.Value{
				"name":  types.StringValue("L2"),
				"tests": types.ListNull(testObjectType()),
				"sections": mustList(t, l3Type, mustObject(t, l3Type, map[string]attr.Value{
					"name":     types.StringValue("L3"),
					"tests":    mustList(t, testObjectType(), testObj(t, "r1", "1.1.1.1")),
					"sections": types.ListNull(sectionObjectType(0)),
				})),
			})),
		})),
	})

	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("exactly three levels must be valid, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsPersonalWithOrganizationScope(t *testing.T) {
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("personal"),
		Visibility:         types.StringValue("Personal"),
		Scope:              types.StringValue(api_client.ScopeOrganization),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           oneSection(t),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Personal + organization must be a config error")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Detail(), api_client.ErrPersonalOrgDetail) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected personal/org diagnostic, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsEmptyDescription(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("n"),
		Description:        types.StringValue(""),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":     types.StringValue("S"),
			"tests":    mustList(t, testObjectType(), testObj(t, "r1", "1.1")),
			"sections": types.ListNull(sectionObjectType(maxSectionDepth - 1)),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, emptyDescriptionMessage) {
		t.Errorf("expected empty-description diagnostic, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsEmptyRootSections(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("empty"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           mustList(t, rootType),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, emptyRootSectionsMessage) {
		t.Errorf("expected empty-root diagnostic, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsEmptyNestedSections(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("empty-nested"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":     types.StringValue("S"),
			"tests":    mustList(t, testObjectType(), testObj(t, "r1", "1.1")),
			"sections": mustList(t, sectionObjectType(maxSectionDepth-1)),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, emptyNestedSectionsMessage) {
		t.Errorf("expected empty-nested-sections diagnostic, got %v", resp.Diagnostics)
	}
	if n := len(resp.Diagnostics); n != 1 {
		t.Errorf("empty nested sections must produce one diagnostic, got %d: %v", n, resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsEmptyTests(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("empty-tests"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":     types.StringValue("S"),
			"tests":    mustList(t, testObjectType()),
			"sections": types.ListNull(sectionObjectType(maxSectionDepth - 1)),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, emptyTestsListMessage) {
		t.Errorf("expected empty-tests diagnostic, got %v", resp.Diagnostics)
	}
	if n := len(resp.Diagnostics); n != 1 {
		t.Errorf("empty tests must produce one diagnostic, got %d: %v", n, resp.Diagnostics)
	}
	if hasDetail(resp, leafNeedsTestMessage) {
		t.Error("must not also report the leaf-needs-test message")
	}
}

func TestValidateConfig_AllowsUnknownTests(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("from-data"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":     types.StringValue("Selected controls"),
			"tests":    types.ListUnknown(testObjectType()),
			"sections": types.ListNull(sectionObjectType(maxSectionDepth - 1)),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unknown tests (data-source for-expression) must wait for plan, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsEmptyLeaf(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("hole"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":     types.StringValue("A"),
			"tests":    types.ListNull(testObjectType()),
			"sections": types.ListNull(sectionObjectType(maxSectionDepth - 1)),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, leafNeedsTestMessage) {
		t.Errorf("expected leaf-needs-test diagnostic, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsEmptyTestsWithNestedSections(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	childType := sectionObjectType(maxSectionDepth - 1)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("empty-tests-nested"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":  types.StringValue("P"),
			"tests": mustList(t, testObjectType()),
			"sections": mustList(t, childType, mustObject(t, childType, map[string]attr.Value{
				"name":     types.StringValue("C"),
				"tests":    mustList(t, testObjectType(), testObj(t, "r1", "1.1.1")),
				"sections": types.ListNull(sectionObjectType(maxSectionDepth - 2)),
			})),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, emptyTestsListMessage) {
		t.Errorf("expected empty-tests diagnostic, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsFourthLevel(t *testing.T) {
	l4 := sectionObjectType(0)
	l3 := sectionObjectType(1)
	l2 := sectionObjectType(2)
	l1 := sectionObjectType(3)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("four"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, l1, mustObject(t, l1, map[string]attr.Value{
			"name":  types.StringValue("A"),
			"tests": types.ListNull(testObjectType()),
			"sections": mustList(t, l2, mustObject(t, l2, map[string]attr.Value{
				"name":  types.StringValue("B"),
				"tests": types.ListNull(testObjectType()),
				"sections": mustList(t, l3, mustObject(t, l3, map[string]attr.Value{
					"name":  types.StringValue("C"),
					"tests": types.ListNull(testObjectType()),
					"sections": mustList(t, l4, mustObject(t, l4, map[string]attr.Value{
						"name": types.StringValue("D"),
					})),
				})),
			})),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, fourthLevelMessage) {
		t.Errorf("expected fourth-level diagnostic, got %v", resp.Diagnostics)
	}
}

func TestSectionIDMatchesDepth(t *testing.T) {
	tests := []struct {
		id, parent string
		depth      int
		ok         bool
	}{
		{"1", "", 1, true},
		{"7", "", 1, true},
		{"1.1", "", 1, false},
		{"CC6", "", 1, false},
		{"1.1", "1", 2, true},
		{"7.2", "7", 2, true},
		{"9", "1", 2, false},
		{"1", "1", 2, false},
		{"CC6", "1", 2, false},
		{"1.1.1", "1", 2, false},
		{"7.2.1", "7.2", 3, true},
		{"7.3.1", "7.2", 3, false},
		{"", "", 1, true},
		{"+7", "", 1, false},
		{"-1", "", 1, false},
		{"07", "", 1, true},
	}
	for _, tt := range tests {
		if got := sectionIDMatchesDepth(tt.id, tt.parent, tt.depth); got != tt.ok {
			t.Errorf("sectionIDMatchesDepth(%q, %q, %d) = %v, want %v", tt.id, tt.parent, tt.depth, got, tt.ok)
		}
	}
}

func TestRuleIDInFrameworkValid(t *testing.T) {
	tests := []struct {
		id    string
		depth int
		ok    bool
	}{
		{"1.1", 1, true},
		{"1.1.1", 1, false},
		{"1.1", 2, false},
		{"1.1.1", 2, true},
		{"1.1.1.1", 3, true},
		{"1.1.1.1", 1, false},
		{"8.8", 1, true},
		{"", 1, true},
		{"5", 1, false},
		{"V-225223", 1, false},
		{"V1.1", 1, false},
		{"1.", 1, false},
		{".1", 1, false},
		{"+1.1", 1, false},
		{"-1.1", 1, false},
	}
	for _, tt := range tests {
		if got := ruleIDInFrameworkValid(tt.id, tt.depth); got != tt.ok {
			t.Errorf("ruleIDInFrameworkValid(%q, %d) = %v, want %v", tt.id, tt.depth, got, tt.ok)
		}
	}
}

func TestValidateConfig_RejectsDotlessRuleIDInFramework(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("dotless"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":     types.StringValue("Alpha"),
			"tests":    mustList(t, testObjectType(), testObj(t, "r1", "5")),
			"sections": types.ListNull(sectionObjectType(maxSectionDepth - 1)),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, invalidRuleIDInFrameworkMessage) {
		t.Errorf("expected illegal rule_id_in_framework diagnostic, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsShallowRuleIDInFramework(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	childType := sectionObjectType(maxSectionDepth - 1)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("shallow"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":  types.StringValue("Parent"),
			"tests": types.ListNull(testObjectType()),
			"sections": mustList(t, childType, mustObject(t, childType, map[string]attr.Value{
				"name":     types.StringValue("Child"),
				"tests":    mustList(t, testObjectType(), testObj(t, "r1", "1.1")),
				"sections": types.ListNull(sectionObjectType(maxSectionDepth - 2)),
			})),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, invalidRuleIDInFrameworkMessage) {
		t.Errorf("expected shallow rule_id_in_framework diagnostic, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_AcceptsExplicitRuleIDNotMatchingSection(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("prefix"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":     types.StringValue("Alpha"),
			"tests":    mustList(t, testObjectType(), testObj(t, "r1", "8.8")),
			"sections": types.ListNull(sectionObjectType(maxSectionDepth - 1)),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("8.8 under an omitted section id must be accepted, got %v", resp.Diagnostics)
	}
}

func TestResolveSiblingIDs(t *testing.T) {
	typ := sectionObjectType(maxSectionDepth)
	tests := []siblingIDCase{
		{"duplicate explicit", "", 1, []string{"7", "7"}, []string{"7", "7"}, []bool{true, true}},
		{"descending explicit", "", 1, []string{"2", "1"}, []string{"2", "1"}, []bool{true, true}},
		{"ascending explicit", "", 1, []string{"1", "2"}, []string{"1", "2"}, []bool{true, true}},
		{"explicit then omitted next above", "7", 2, []string{"7.2", ""}, []string{"7.2", "7.3"}, []bool{true, true}},
		{"all omitted", "", 1, []string{"", ""}, []string{"1", "2"}, []bool{true, true}},
		{"invalid keeps user value", "", 1, []string{"CC6"}, []string{"CC6"}, []bool{false}},
		{"invalid nested keeps user value", "1", 2, []string{"9"}, []string{"9"}, []bool{false}},
		{"dotted top-level keeps user value", "", 1, []string{"1.1"}, []string{"1.1"}, []bool{false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertResolvedSiblingIDs(t, typ, tt)
		})
	}
}

func TestValidateConfig_RejectsDuplicateSiblingSectionIDs(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("dup"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType,
			siblingLeaf(t, rootType, "Alpha", "7", "r1"),
			siblingLeaf(t, rootType, "Beta", "7", "r2"),
		),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, duplicateSectionIDMessage) {
		t.Errorf("expected duplicate sibling diagnostic, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsDescendingSiblingSectionIDs(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("desc"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType,
			siblingLeaf(t, rootType, "Alpha", "2", "r1"),
			siblingLeaf(t, rootType, "Beta", "1", "r2"),
		),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, duplicateSectionIDMessage) {
		t.Errorf("expected descending sibling diagnostic, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_AcceptsAscendingSiblingSectionIDs(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("asc"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType,
			siblingLeaf(t, rootType, "Alpha", "1", "r1"),
			siblingLeaf(t, rootType, "Beta", "2", "r2"),
		),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("1 then 2 must be valid, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_AcceptsOmittedSiblingAfterExplicit(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	childType := sectionObjectType(maxSectionDepth - 1)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("skip"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":                    types.StringValue("Parent"),
			"section_id_in_framework": types.StringValue("7"),
			"tests":                   types.ListNull(testObjectType()),
			"sections": mustList(t, childType,
				mustObject(t, childType, map[string]attr.Value{
					"name":                    types.StringValue("C1"),
					"section_id_in_framework": types.StringValue("7.2"),
					"tests":                   mustList(t, testObjectType(), testObj(t, "r1", "")),
					"sections":                types.ListNull(sectionObjectType(maxSectionDepth - 2)),
				}),
				mustObject(t, childType, map[string]attr.Value{
					"name":     types.StringValue("C2"),
					"tests":    mustList(t, testObjectType(), testObj(t, "r2", "")),
					"sections": types.ListNull(sectionObjectType(maxSectionDepth - 2)),
				}),
			),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("omitted sibling must become 7.3 after 7.2, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsDottedTopLevelSectionID(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("dotted"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":                    types.StringValue("Alpha"),
			"section_id_in_framework": types.StringValue("1.1"),
			"tests":                   mustList(t, testObjectType(), testObj(t, "r1", "1.1")),
			"sections":                types.ListNull(sectionObjectType(maxSectionDepth - 1)),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, invalidSectionIDMessage) {
		t.Errorf("expected section-id diagnostic, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsNestedSectionIDNotExtendingParent(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	childType := sectionObjectType(maxSectionDepth - 1)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("mismatch"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":                    types.StringValue("Parent"),
			"section_id_in_framework": types.StringValue("1"),
			"tests":                   types.ListNull(testObjectType()),
			"sections": mustList(t, childType, mustObject(t, childType, map[string]attr.Value{
				"name":                    types.StringValue("Child"),
				"section_id_in_framework": types.StringValue("9"),
				"tests":                   mustList(t, testObjectType(), testObj(t, "r1", "9.1.1")),
				"sections":                types.ListNull(sectionObjectType(maxSectionDepth - 2)),
			})),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, invalidSectionIDMessage) {
		t.Errorf("expected section-id diagnostic, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_AcceptsNestedSectionIDExtendingParent(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	childType := sectionObjectType(maxSectionDepth - 1)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("ok"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":                    types.StringValue("Parent"),
			"section_id_in_framework": types.StringValue("7"),
			"tests":                   types.ListNull(testObjectType()),
			"sections": mustList(t, childType, mustObject(t, childType, map[string]attr.Value{
				"name":                    types.StringValue("Child"),
				"section_id_in_framework": types.StringValue("7.2"),
				"tests":                   mustList(t, testObjectType(), testObj(t, "r1", "")),
				"sections":                types.ListNull(sectionObjectType(maxSectionDepth - 2)),
			})),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("7 / 7.2 must be valid, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_RejectsNonNumericSectionID(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("cc6"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":                    types.StringValue("A"),
			"section_id_in_framework": types.StringValue("CC6"),
			"tests":                   mustList(t, testObjectType(), testObj(t, "r1", "1.1")),
			"sections":                types.ListNull(sectionObjectType(maxSectionDepth - 1)),
		})),
	})
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !hasDetail(resp, invalidSectionIDMessage) {
		t.Errorf("expected invalid section id diagnostic, got %v", resp.Diagnostics)
	}
}
