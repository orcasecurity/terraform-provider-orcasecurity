package custom_compliance_framework

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func oneSection(t *testing.T) types.List {
	t.Helper()
	rootType := sectionObjectType(maxSectionDepth)
	return mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
		"name":     types.StringValue("Flat"),
		"tests":    mustList(t, testObjectType(), testObj(t, "r1", "1.1")),
		"sections": types.ListNull(sectionObjectType(maxSectionDepth - 1)),
	}))
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

func nestedChild(t *testing.T, name, id string, tests ...types.Object) types.Object {
	t.Helper()
	childType := sectionObjectType(maxSectionDepth - 1)
	attrs := map[string]attr.Value{
		"name":     types.StringValue(name),
		"tests":    mustList(t, testObjectType(), testsAsValues(tests)...),
		"sections": types.ListNull(sectionObjectType(maxSectionDepth - 2)),
	}
	if id != "" {
		attrs["section_id_in_framework"] = types.StringValue(id)
	}
	return mustObject(t, childType, attrs)
}

func nestedParent(t *testing.T, name, id string, children ...types.Object) types.Object {
	t.Helper()
	rootType := sectionObjectType(maxSectionDepth)
	childType := sectionObjectType(maxSectionDepth - 1)
	vals := make([]attr.Value, len(children))
	for i, c := range children {
		vals[i] = c
	}
	attrs := map[string]attr.Value{
		"name":     types.StringValue(name),
		"tests":    types.ListNull(testObjectType()),
		"sections": mustList(t, childType, vals...),
	}
	if id != "" {
		attrs["section_id_in_framework"] = types.StringValue(id)
	}
	return mustObject(t, rootType, attrs)
}

func sectionAt(t *testing.T, remainingDepth int, name, id string, tests []types.Object, children []types.Object) types.Object {
	t.Helper()
	typ := sectionObjectType(remainingDepth)
	attrs := map[string]attr.Value{
		"name": types.StringValue(name),
	}
	if id != "" {
		attrs["section_id_in_framework"] = types.StringValue(id)
	}
	if len(tests) > 0 {
		attrs["tests"] = mustList(t, testObjectType(), testsAsValues(tests)...)
	} else {
		attrs["tests"] = types.ListNull(testObjectType())
	}
	if remainingDepth > 0 {
		childType := sectionObjectType(remainingDepth - 1)
		if len(children) > 0 {
			vals := make([]attr.Value, len(children))
			for i, c := range children {
				vals[i] = c
			}
			attrs["sections"] = mustList(t, childType, vals...)
		} else {
			attrs["sections"] = types.ListNull(childType)
		}
	}
	return mustObject(t, typ, attrs)
}

func frameworkLists(t *testing.T, sections types.List) customComplianceFrameworkResourceModel {
	t.Helper()
	return customComplianceFrameworkResourceModel{
		ID:                 types.StringValue("1"),
		Name:               types.StringValue("n"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections:           sections,
	}
}

func ptrModel(m customComplianceFrameworkResourceModel) *customComplianceFrameworkResourceModel {
	return &m
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

type siblingIDCase struct {
	name   string
	parent string
	depth  int
	ids    []string
	want   []string
	valid  []bool
}

func assertResolvedSiblingIDs(t *testing.T, typ types.ObjectType, tt siblingIDCase) {
	t.Helper()
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
}
