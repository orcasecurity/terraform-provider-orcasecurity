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
	if _, ok := l4.NestedObject.Attributes["tests"]; ok {
		t.Fatal("rejected level must not document tests")
	}
	if _, ok := l4.NestedObject.Attributes["section_id_in_framework"]; ok {
		t.Fatal("rejected level must not document section_id_in_framework")
	}
	if _, ok := l4.NestedObject.Attributes["name"]; !ok {
		t.Fatal("rejected level keeps name so Terraform cannot silently discard the object")
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
	r, req := planStateConfig(t, state, plan, nil)
	return r, req
}

func planStateConfig(t *testing.T, state, plan customComplianceFrameworkResourceModel, config *customComplianceFrameworkResourceModel) (*customComplianceFrameworkResource, resource.ModifyPlanRequest) {
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
	req := resource.ModifyPlanRequest{State: st, Plan: pl}
	if config != nil {
		cs := tfsdk.State{Schema: schemaResp.Schema}
		if diags := cs.Set(context.Background(), config); diags.HasError() {
			t.Fatalf("config: %v", diags)
		}
		req.Config = tfsdk.Config{Schema: schemaResp.Schema, Raw: cs.Raw}
	}
	return r, req
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

func flatSection(t *testing.T, name, id string, tests ...types.Object) types.Object {
	t.Helper()
	rootType := sectionObjectType(maxSectionDepth)
	attrs := map[string]attr.Value{
		"name":     types.StringValue(name),
		"tests":    mustList(t, testObjectType(), testsAsValues(tests)...),
		"sections": types.ListNull(sectionObjectType(maxSectionDepth - 1)),
	}
	if id != "" {
		attrs["section_id_in_framework"] = types.StringValue(id)
	}
	return mustObject(t, rootType, attrs)
}

func testsAsValues(objs []types.Object) []attr.Value {
	out := make([]attr.Value, len(objs))
	for i, o := range objs {
		out[i] = o
	}
	return out
}

func testComputed(t *testing.T, ruleID, rid, priority, origin string) types.Object {
	t.Helper()
	str := func(s string) types.String {
		if s == "" {
			return types.StringNull()
		}
		return types.StringValue(s)
	}
	return mustObject(t, testObjectType(), map[string]attr.Value{
		"rule_id":              types.StringValue(ruleID),
		"rule_id_in_framework": str(rid),
		"priority":             str(priority),
		"control_unique_id":    types.StringNull(),
		"origin_framework_id":  str(origin),
	})
}

func TestModifyPlan_NewTestMarksComputedUnknown(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	state := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           mustList(t, rootType, flatSection(t, "Alpha", "1", testComputed(t, "r1", "1.1", "Medium", "9"))),
	}
	plan := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, flatSection(t, "Alpha", "1",
			testComputed(t, "r1", "1.1", "Medium", "9"),
			testComputed(t, "r2", "", "", ""),
		)),
	}
	config := plan
	r, req := planStateConfig(t, state, plan, &config)
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var out customComplianceFrameworkResourceModel
	if diags := resp.Plan.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	sec := out.Sections.Elements()[0].(types.Object)
	tests := sec.Attributes()["tests"].(types.List)
	first := tests.Elements()[0].(types.Object).Attributes()
	if first["priority"].IsUnknown() {
		t.Error("existing test priority must stay known")
	}
	second := tests.Elements()[1].(types.Object).Attributes()
	for _, name := range computedTestAttrs {
		if !second[name].IsUnknown() {
			t.Errorf("new test %s must be unknown, got %#v", name, second[name])
		}
	}
}

func TestModifyPlan_SectionRenumberMarksRuleIDUnknown(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	state := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           mustList(t, rootType, flatSection(t, "Alpha", "1", testComputed(t, "r1", "1.1", "Medium", "9"))),
	}
	plan := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           mustList(t, rootType, flatSection(t, "Alpha", "5", testComputed(t, "r1", "1.1", "Medium", "9"))),
	}
	config := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           mustList(t, rootType, flatSection(t, "Alpha", "5", testComputed(t, "r1", "", "", ""))),
	}
	r, req := planStateConfig(t, state, plan, &config)
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var out customComplianceFrameworkResourceModel
	if diags := resp.Plan.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	sec := out.Sections.Elements()[0].(types.Object)
	test0 := sec.Attributes()["tests"].(types.List).Elements()[0].(types.Object).Attributes()
	if !test0["rule_id_in_framework"].IsUnknown() {
		t.Errorf("omitted rule_id_in_framework must re-derive after section renumber, got %#v", test0["rule_id_in_framework"])
	}
	if test0["priority"].IsUnknown() {
		t.Error("priority must stay known when only the section id changed")
	}
}

func TestModifyPlan_EarlierSiblingIDUnpinsLaterOmitted(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	state := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType,
			flatSection(t, "Alpha", "1", testComputed(t, "r1", "1.1", "Medium", "9")),
			flatSection(t, "Beta", "2", testComputed(t, "r2", "2.1", "Medium", "9")),
		),
	}
	plan := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType,
			flatSection(t, "Alpha", "5", testComputed(t, "r1", "1.1", "Medium", "9")),
			flatSection(t, "Beta", "2", testComputed(t, "r2", "2.1", "Medium", "9")),
		),
	}
	config := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType,
			flatSection(t, "Alpha", "5", testComputed(t, "r1", "", "", "")),
			flatSection(t, "Beta", "", testComputed(t, "r2", "", "", "")),
		),
	}
	r, req := planStateConfig(t, state, plan, &config)
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var out customComplianceFrameworkResourceModel
	if diags := resp.Plan.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	beta := out.Sections.Elements()[1].(types.Object).Attributes()
	if !beta["section_id_in_framework"].IsUnknown() {
		t.Errorf("omitted later sibling id must unpin when an earlier id changes, got %#v", beta["section_id_in_framework"])
	}
	test0 := beta["tests"].(types.List).Elements()[0].(types.Object).Attributes()
	if !test0["rule_id_in_framework"].IsUnknown() {
		t.Errorf("omitted later sibling control id must re-derive, got %#v", test0["rule_id_in_framework"])
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
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "section_id_in_framework") {
		t.Errorf("write payload must not send the ignored section_id_in_framework key, got %s", raw)
	}
}

func TestRequestFromPlanUsesSectionIDAsControlPrefix(t *testing.T) {
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
	if req.Sections[0].Tests[0].RuleIDInFramework != "7.1" {
		t.Errorf("derived control id must use custom section id, got %q", req.Sections[0].Tests[0].RuleIDInFramework)
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "section_id_in_framework") {
		t.Errorf("write payload must not send the ignored section_id_in_framework key, got %s", raw)
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

func siblingLeaf(t *testing.T, typ types.ObjectType, name, id, rule string) types.Object {
	t.Helper()
	attrs := map[string]attr.Value{
		"name":     types.StringValue(name),
		"tests":    mustList(t, testObjectType(), testObj(t, rule, "")),
		"sections": types.ListNull(sectionObjectType(maxSectionDepth - 1)),
	}
	if id != "" {
		attrs["section_id_in_framework"] = types.StringValue(id)
	}
	return mustObject(t, typ, attrs)
}

func TestResolveSiblingIDs(t *testing.T) {
	typ := sectionObjectType(maxSectionDepth)
	tests := []struct {
		name   string
		parent string
		depth  int
		ids    []string
		want   []string
		valid  []bool
	}{
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
			elems := make([]attr.Value, len(tt.ids))
			for i, id := range tt.ids {
				elems[i] = siblingLeaf(t, typ, "S", id, "r1")
			}
			got := resolveSiblingIDs(elems, tt.parent, tt.depth)
			if len(got) != len(tt.want) {
				t.Fatalf("len=%d want %d", len(got), len(tt.want))
			}
			for i, w := range tt.want {
				if got[i].ID != w {
					t.Errorf("id[%d]=%q want %q", i, got[i].ID, w)
				}
				if got[i].Valid != tt.valid[i] {
					t.Errorf("valid[%d]=%v want %v", i, got[i].Valid, tt.valid[i])
				}
			}
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

func TestRequestFromPlanOmittedSiblingFollowsPrevious(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	childType := sectionObjectType(maxSectionDepth - 1)
	sections := mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
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
	}))
	req, diags := requestFromPlan(context.Background(), customComplianceFrameworkResourceModel{
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           sections,
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if req.Sections[0].Sections[0].Tests[0].RuleIDInFramework != "7.2.1" {
		t.Errorf("explicit 7.2: %+v", req.Sections[0].Sections[0].Tests)
	}
	if req.Sections[0].Sections[1].Tests[0].RuleIDInFramework != "7.3.1" {
		t.Errorf("omitted sibling must become 7.3, got %+v", req.Sections[0].Sections[1].Tests)
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
				"tests":                   mustList(t, testObjectType(), testObj(t, "r1", "9.1")),
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
