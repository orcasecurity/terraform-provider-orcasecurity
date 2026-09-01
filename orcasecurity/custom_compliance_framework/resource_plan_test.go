package custom_compliance_framework

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

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

func TestModifyPlan_ParentIDChangeUnpinsOmittedChildren(t *testing.T) {
	rootType := sectionObjectType(maxSectionDepth)
	state := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, nestedParent(t, "Parent", "7",
			nestedChild(t, "C1", "7.1", testComputed(t, "r1", "7.1.1", "Medium", "9")),
			nestedChild(t, "C2", "7.2", testComputed(t, "r2", "7.2.1", "Medium", "9")),
		)),
	}
	plan := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, nestedParent(t, "Parent", "9",
			nestedChild(t, "C1", "7.1", testComputed(t, "r1", "7.1.1", "Medium", "9")),
			nestedChild(t, "C2", "7.2", testComputed(t, "r2", "7.2.1", "Medium", "9")),
		)),
	}
	config := customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, nestedParent(t, "Parent", "9",
			nestedChild(t, "C1", "", testComputed(t, "r1", "", "", "")),
			nestedChild(t, "C2", "", testComputed(t, "r2", "", "", "")),
		)),
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
	parent := out.Sections.Elements()[0].(types.Object)
	children := parent.Attributes()["sections"].(types.List)
	for i, e := range children.Elements() {
		ch := e.(types.Object).Attributes()
		if !ch["section_id_in_framework"].IsUnknown() {
			t.Errorf("child[%d] section id must unpin when parent id changes, got %#v", i, ch["section_id_in_framework"])
		}
		rid := ch["tests"].(types.List).Elements()[0].(types.Object).Attributes()["rule_id_in_framework"]
		if !rid.IsUnknown() {
			t.Errorf("child[%d] control id must re-derive when parent id changes, got %#v", i, rid)
		}
	}
}

func TestModifyPlan_GrandparentIDChangeUnpinsOmittedTree(t *testing.T) {
	root := maxSectionDepth
	mid := maxSectionDepth - 1
	leaf := maxSectionDepth - 2
	stateTree := sectionAt(t, root, "GP", "7", nil, []types.Object{
		sectionAt(t, mid, "P", "7.1", nil, []types.Object{
			sectionAt(t, leaf, "C1", "7.1.1", []types.Object{testComputed(t, "r1", "7.1.1.1", "Medium", "9")}, nil),
			sectionAt(t, leaf, "C2", "7.1.2", []types.Object{testComputed(t, "r2", "7.1.2.1", "Medium", "9")}, nil),
		}),
	})
	planTree := sectionAt(t, root, "GP", "9", nil, []types.Object{
		sectionAt(t, mid, "P", "7.1", nil, []types.Object{
			sectionAt(t, leaf, "C1", "7.1.1", []types.Object{testComputed(t, "r1", "7.1.1.1", "Medium", "9")}, nil),
			sectionAt(t, leaf, "C2", "7.1.2", []types.Object{testComputed(t, "r2", "7.1.2.1", "Medium", "9")}, nil),
		}),
	})
	configTree := sectionAt(t, root, "GP", "9", nil, []types.Object{
		sectionAt(t, mid, "P", "", nil, []types.Object{
			sectionAt(t, leaf, "C1", "", []types.Object{testComputed(t, "r1", "", "", "")}, nil),
			sectionAt(t, leaf, "C2", "", []types.Object{testComputed(t, "r2", "", "", "")}, nil),
		}),
	})
	rootType := sectionObjectType(root)
	r, req := planStateConfig(t,
		frameworkLists(t, mustList(t, rootType, stateTree)),
		frameworkLists(t, mustList(t, rootType, planTree)),
		ptrModel(frameworkLists(t, mustList(t, rootType, configTree))),
	)
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var out customComplianceFrameworkResourceModel
	if diags := resp.Plan.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	parent := out.Sections.Elements()[0].(types.Object).Attributes()["sections"].(types.List).Elements()[0].(types.Object)
	if !parent.Attributes()["section_id_in_framework"].IsUnknown() {
		t.Errorf("omitted child of a renumbered grandparent must unpin, got %#v", parent.Attributes()["section_id_in_framework"])
	}
	for i, e := range parent.Attributes()["sections"].(types.List).Elements() {
		ch := e.(types.Object).Attributes()
		if !ch["section_id_in_framework"].IsUnknown() {
			t.Errorf("grandchild[%d] section id must unpin, got %#v", i, ch["section_id_in_framework"])
		}
		rid := ch["tests"].(types.List).Elements()[0].(types.Object).Attributes()["rule_id_in_framework"]
		if !rid.IsUnknown() {
			t.Errorf("grandchild[%d] control id must re-derive, got %#v", i, rid)
		}
	}
}

func TestModifyPlan_SiblingIDChangeUnpinsNestedSubtree(t *testing.T) {
	root := maxSectionDepth
	child := maxSectionDepth - 1
	state := mustList(t, sectionObjectType(root),
		flatSection(t, "A", "1", testComputed(t, "r1", "1.1", "Medium", "9")),
		sectionAt(t, root, "B", "2", nil, []types.Object{
			sectionAt(t, child, "C1", "2.1", []types.Object{testComputed(t, "r2", "2.1.1", "Medium", "9")}, nil),
			sectionAt(t, child, "C2", "2.2", []types.Object{testComputed(t, "r3", "2.2.1", "Medium", "9")}, nil),
		}),
	)
	plan := mustList(t, sectionObjectType(root),
		flatSection(t, "A", "5", testComputed(t, "r1", "1.1", "Medium", "9")),
		sectionAt(t, root, "B", "2", nil, []types.Object{
			sectionAt(t, child, "C1", "2.1", []types.Object{testComputed(t, "r2", "2.1.1", "Medium", "9")}, nil),
			sectionAt(t, child, "C2", "2.2", []types.Object{testComputed(t, "r3", "2.2.1", "Medium", "9")}, nil),
		}),
	)
	config := mustList(t, sectionObjectType(root),
		flatSection(t, "A", "5", testComputed(t, "r1", "", "", "")),
		sectionAt(t, root, "B", "", nil, []types.Object{
			sectionAt(t, child, "C1", "", []types.Object{testComputed(t, "r2", "", "", "")}, nil),
			sectionAt(t, child, "C2", "", []types.Object{testComputed(t, "r3", "", "", "")}, nil),
		}),
	)
	r, req := planStateConfig(t, frameworkLists(t, state), frameworkLists(t, plan), ptrModel(frameworkLists(t, config)))
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var out customComplianceFrameworkResourceModel
	if diags := resp.Plan.Get(context.Background(), &out); diags.HasError() {
		t.Fatal(diags)
	}
	b := out.Sections.Elements()[1].(types.Object)
	if !b.Attributes()["section_id_in_framework"].IsUnknown() {
		t.Errorf("omitted sibling B must unpin when A changes, got %#v", b.Attributes()["section_id_in_framework"])
	}
	for i, e := range b.Attributes()["sections"].(types.List).Elements() {
		ch := e.(types.Object).Attributes()
		if !ch["section_id_in_framework"].IsUnknown() {
			t.Errorf("B.child[%d] section id must unpin with sibling A, got %#v", i, ch["section_id_in_framework"])
		}
		rid := ch["tests"].(types.List).Elements()[0].(types.Object).Attributes()["rule_id_in_framework"]
		if !rid.IsUnknown() {
			t.Errorf("B.child[%d] control id must re-derive, got %#v", i, rid)
		}
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
