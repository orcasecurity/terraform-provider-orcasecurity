package custom_compliance_framework

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func schemaAndConfig(t *testing.T, model customComplianceFrameworkResourceModel) (*customComplianceFrameworkResource, tfsdk.Config) {
	t.Helper()
	r := &customComplianceFrameworkResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	st := tfsdk.State{Schema: schemaResp.Schema}
	if diags := st.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set config: %v", diags)
	}
	return r, tfsdk.Config{Schema: schemaResp.Schema, Raw: st.Raw}
}

func mustObject(t *testing.T, typ types.ObjectType, attrs map[string]attr.Value) types.Object {
	t.Helper()
	if _, ok := typ.AttrTypes["section_id_in_framework"]; ok {
		if _, present := attrs["section_id_in_framework"]; !present {
			attrs["section_id_in_framework"] = types.StringNull()
		}
	}
	obj, d := types.ObjectValue(typ.AttrTypes, attrs)
	if d.HasError() {
		t.Fatal(d)
	}
	return obj
}

func mustList(t *testing.T, elem attr.Type, vals ...attr.Value) types.List {
	t.Helper()
	l, d := types.ListValue(elem, vals)
	if d.HasError() {
		t.Fatal(d)
	}
	return l
}

func testObj(t *testing.T, ruleID, rid string) types.Object {
	t.Helper()
	typ := testObjectType()
	return mustObject(t, typ, map[string]attr.Value{
		"rule_id":              types.StringValue(ruleID),
		"rule_id_in_framework": types.StringValue(rid),
		"priority":             types.StringNull(),
		"control_unique_id":    types.StringNull(),
		"origin_framework_id":  types.StringNull(),
	})
}

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

func TestSchema_FourthLevelExistsForRejection(t *testing.T) {
	r := &customComplianceFrameworkResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	l1, ok := schemaResp.Schema.Attributes["sections"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("sections")
	}
	if len(l1.Validators) != 0 {
		t.Errorf("root sections must not duplicate ValidateConfig with SizeAtLeast, got %d validators", len(l1.Validators))
	}
	l2, ok := l1.NestedObject.Attributes["sections"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("sections.sections")
	}
	l3, ok := l2.NestedObject.Attributes["sections"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("sections.sections.sections")
	}
	l4, ok := l3.NestedObject.Attributes["sections"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("fourth nested sections must be in the schema so Terraform does not discard it")
	}
	if _, ok := l4.NestedObject.Attributes["sections"]; ok {
		t.Fatal("fifth nested sections must not be in the schema")
	}
	if _, ok := l4.NestedObject.Attributes["section_id_in_framework"]; !ok {
		t.Fatal("section_id_in_framework must exist at every section level")
	}
	if !strings.Contains(l4.Description, "Not supported") {
		t.Errorf("fourth nested sections must advertise rejection, got %q", l4.Description)
	}
	if strings.Contains(l1.NestedObject.Attributes["sections"].(schema.ListNestedAttribute).Description, "Not supported") {
		t.Error("supported nesting levels must not use the rejected-level description")
	}
	if strings.Contains(l2.Description, "fourth nested") {
		t.Errorf("level-2 nested sections must not mention a fourth level, got %q", l2.Description)
	}
	tests, ok := l1.NestedObject.Attributes["tests"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("tests")
	}
	if _, ok := tests.NestedObject.Attributes["reference_id"]; ok {
		t.Fatal("reference_id must not be a test attribute")
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
					"tests":    mustList(t, testObjectType(), testObj(t, "r1", "1.1.1")),
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

func TestSchema_LoosenedAndAdditive(t *testing.T) {
	r := &customComplianceFrameworkResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	attrs := schemaResp.Schema.Attributes

	desc, ok := attrs["description"].(schema.StringAttribute)
	if !ok || !desc.Optional || desc.Required {
		t.Errorf("description must be Optional, got %#v", attrs["description"])
	}
	if _, ok := attrs["visibility"]; !ok {
		t.Error("missing visibility")
	}
	if _, ok := attrs["scope"]; !ok {
		t.Error("missing scope")
	}
	if _, ok := attrs["forced_cloud_vendors"]; !ok {
		t.Error("missing forced_cloud_vendors")
	}

	sections, ok := attrs["sections"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("sections must be ListNestedAttribute")
	}
	tests, ok := sections.NestedObject.Attributes["tests"].(schema.ListNestedAttribute)
	if !ok || !tests.Optional || tests.Required {
		t.Errorf("sections.tests must be Optional, got %#v", sections.NestedObject.Attributes["tests"])
	}
	if _, ok := sections.NestedObject.Attributes["sections"]; !ok {
		t.Error("missing nested sections")
	}
	if _, ok := sections.NestedObject.Attributes["section_id_in_framework"]; !ok {
		t.Error("missing section_id_in_framework")
	}

	scope, ok := attrs["scope"].(schema.StringAttribute)
	if !ok {
		t.Fatal("scope")
	}
	want := stringplanmodifier.RequiresReplace().Description(context.Background())
	foundReplace := false
	for _, m := range scope.PlanModifiers {
		if m.Description(context.Background()) == want {
			foundReplace = true
		}
	}
	if !foundReplace {
		t.Error("scope must RequiresReplace")
	}
}

func oneSection(t *testing.T) types.List {
	t.Helper()
	rootType := sectionObjectType(maxSectionDepth)
	return mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
		"name":     types.StringValue("Flat"),
		"tests":    mustList(t, testObjectType(), testObj(t, "r1", "1.1")),
		"sections": types.ListNull(sectionObjectType(maxSectionDepth - 1)),
	}))
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

func TestUpdateRequestOmitsEmptyForcedCloudVendors(t *testing.T) {
	empty, d := types.SetValue(types.StringType, []attr.Value{})
	if d.HasError() {
		t.Fatal(d)
	}
	req, diags := requestFromPlan(context.Background(), customComplianceFrameworkResourceModel{
		Name:               types.StringValue("n"),
		ForcedCloudVendors: empty,
		Sections:           oneSection(t),
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "forced_cloud_vendors") {
		t.Errorf("empty set must omit forced_cloud_vendors, got %s", raw)
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
				"tests":    mustList(t, testObjectType(), testObj(t, "r1", "1.1")),
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
						"name":  types.StringValue("D"),
						"tests": mustList(t, testObjectType(), testObj(t, "r1", "1.1.1.1")),
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

func hasDetail(resp *resource.ValidateConfigResponse, want string) bool {
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Detail(), want) {
			return true
		}
	}
	return false
}

func planAndState(t *testing.T, state, plan customComplianceFrameworkResourceModel) (*customComplianceFrameworkResource, resource.ModifyPlanRequest) {
	t.Helper()
	r := &customComplianceFrameworkResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	st := tfsdk.State{Schema: schemaResp.Schema}
	if diags := st.Set(context.Background(), &state); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	pl := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := pl.Set(context.Background(), &plan); diags.HasError() {
		t.Fatalf("plan: %v", diags)
	}
	return r, resource.ModifyPlanRequest{State: st, Plan: pl}
}

func TestModifyPlan_RejectsOrganizationalToPersonal(t *testing.T) {
	state := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		Visibility:         types.StringValue(api_client.VisibilityOrganizational),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           oneSection(t),
	}
	plan := state
	plan.Visibility = types.StringValue(api_client.VisibilityPersonal)
	r, req := planAndState(t, state, plan)
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Organizational → Personal must be a plan error")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Detail(), api_client.ErrVisibilityDowngradeDetail) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected downgrade diagnostic, got %v", resp.Diagnostics)
	}
}

func TestModifyPlan_AllowsPersonalToOrganizational(t *testing.T) {
	state := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		Visibility:         types.StringValue(api_client.VisibilityPersonal),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           oneSection(t),
	}
	plan := state
	plan.Visibility = types.StringValue(api_client.VisibilityOrganizational)
	r, req := planAndState(t, state, plan)
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Personal → Organizational is a promotion, got %v", resp.Diagnostics)
	}
}

func TestRequestFromPlanDerivesOmittedRuleIDInFramework(t *testing.T) {
	typ := testObjectType()
	rootType := sectionObjectType(maxSectionDepth)
	sections := mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
		"name": types.StringValue("Flat"),
		"tests": mustList(t, typ, mustObject(t, typ, map[string]attr.Value{
			"rule_id":              types.StringValue("r1"),
			"rule_id_in_framework": types.StringNull(),
			"priority":             types.StringNull(),
			"control_unique_id":    types.StringNull(),
			"origin_framework_id":  types.StringNull(),
		})),
		"sections": types.ListNull(sectionObjectType(maxSectionDepth - 1)),
	}))
	req, diags := requestFromPlan(context.Background(), customComplianceFrameworkResourceModel{
		Name:               types.StringValue("derived"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           sections,
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if req.Sections[0].Tests[0].RuleIDInFramework != "1.1" {
		t.Errorf("omitted rule_id_in_framework must derive, got %+v", req.Sections[0].Tests)
	}
	if req.Sections[0].SectionIDInFramework != nil {
		t.Errorf("omitted section_id_in_framework must not be sent, got %v", *req.Sections[0].SectionIDInFramework)
	}
}

func TestRequestFromPlanSendsNumericSectionID(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	sections := mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
		"name":                    types.StringValue("Alpha"),
		"section_id_in_framework": types.StringValue("7"),
		"tests":                   mustList(t, testObjectType(), testObj(t, "r1", "")),
		"sections":                types.ListNull(sectionObjectType(maxSectionDepth - 1)),
	}))
	req, diags := requestFromPlan(context.Background(), customComplianceFrameworkResourceModel{
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           sections,
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if req.Sections[0].SectionIDInFramework == nil || *req.Sections[0].SectionIDInFramework != 7 {
		t.Errorf("section_id_in_framework: %v", req.Sections[0].SectionIDInFramework)
	}
	if req.Sections[0].Tests[0].RuleIDInFramework != "7.1" {
		t.Errorf("derived control id must use custom section id, got %q", req.Sections[0].Tests[0].RuleIDInFramework)
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"section_id_in_framework":7`) {
		t.Errorf("write payload must send a JSON number, got %s", raw)
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

func TestSchema_SectionDepthMatchesCatalogConvention(t *testing.T) {
	if schemaSectionDepth != maxSectionDepth+1 {
		t.Fatalf("schemaSectionDepth=%d maxSectionDepth=%d", schemaSectionDepth, maxSectionDepth)
	}
	r := &customComplianceFrameworkResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	root := schemaResp.Schema.Attributes["sections"].(schema.ListNestedAttribute)
	if _, ok := root.NestedObject.Attributes["sections"]; !ok {
		t.Fatal("root remaining depth must include nested sections")
	}
}

func TestRequestFromPlanIncludesScope(t *testing.T) {
	req, diags := requestFromPlan(context.Background(), customComplianceFrameworkResourceModel{
		Name:               types.StringValue("n"),
		Scope:              types.StringValue(api_client.ScopeUser),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           oneSection(t),
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if req.Scope != api_client.ScopeUser {
		t.Errorf("scope: %q", req.Scope)
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"scope":"user"`) {
		t.Errorf("PUT/POST payload must include scope, got %s", raw)
	}
}

func TestModifyPlan_DestroyDoesNotRejectPersonal(t *testing.T) {
	state := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		Visibility:         types.StringValue(api_client.VisibilityPersonal),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           oneSection(t),
	}
	r := &customComplianceFrameworkResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	st := tfsdk.State{Schema: schemaResp.Schema}
	if diags := st.Set(context.Background(), &state); diags.HasError() {
		t.Fatal(diags)
	}
	nullPlan := tfsdk.Plan{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), nil),
	}
	resp := &resource.ModifyPlanResponse{Plan: nullPlan}
	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{State: st, Plan: nullPlan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("destroy must not raise visibility downgrade, got %v", resp.Diagnostics)
	}
}
