package custom_compliance_framework

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	rootType := sectionObjectType(maxSectionDepth - 1)
	childType := sectionObjectType(maxSectionDepth - 2)
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("mixed"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: mustList(t, rootType, mustObject(t, rootType, map[string]attr.Value{
			"name":  types.StringValue("Sec A"),
			"tests": mustList(t, testObjectType(), testObj(t, "r1", "1.1")),
			"sections": mustList(t, childType, mustObject(t, childType, map[string]attr.Value{
				"name":     types.StringValue("Sub A1"),
				"tests":    mustList(t, testObjectType(), testObj(t, "r2", "2.1")),
				"sections": types.ListNull(sectionObjectType(0)),
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

func TestSchema_StopsAtThreeLevels(t *testing.T) {
	r := &customComplianceFrameworkResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	l1, ok := schemaResp.Schema.Attributes["sections"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("sections")
	}
	l2, ok := l1.NestedObject.Attributes["sections"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("sections.sections")
	}
	l3, ok := l2.NestedObject.Attributes["sections"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("sections.sections.sections")
	}
	if _, ok := l3.NestedObject.Attributes["sections"]; ok {
		t.Fatal("fourth nested sections must not be in the schema")
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
	l3Type := sectionObjectType(0)
	l2Type := sectionObjectType(1)
	l1Type := sectionObjectType(2)
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
					"name":  types.StringValue("L3"),
					"tests": mustList(t, testObjectType(), testObj(t, "r1", "1.1.1")),
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
}
