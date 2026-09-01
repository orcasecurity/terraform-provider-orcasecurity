package custom_compliance_framework

import (
	"context"
	"strings"
	"testing"

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

func TestValidateConfig_RejectsMixedTestsAndSections(t *testing.T) {
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("mixed"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: []sectionModel{{
			Name: types.StringValue("Sec A"),
			Tests: []testModel{
				{RuleID: types.StringValue("r1"), RuleIDInFramework: types.StringValue("1.1")},
			},
			Sections: []midSectionModel{{
				Name: types.StringValue("Sub A1"),
				Tests: []testModel{
					{RuleID: types.StringValue("r2"), RuleIDInFramework: types.StringValue("2.1")},
				},
			}},
		}},
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

func TestValidateConfig_RejectsFourthLevel(t *testing.T) {
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("too-deep"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: []sectionModel{{
			Name: types.StringValue("L1"),
			Sections: []midSectionModel{{
				Name: types.StringValue("L2"),
				Sections: []leafSectionModel{{
					Name: types.StringValue("L3"),
					Sections: []tooDeepSectionModel{{
						Name: types.StringValue("L4"),
						Tests: []testModel{
							{RuleID: types.StringValue("r1"), RuleIDInFramework: types.StringValue("1.1.1.1")},
						},
					}},
				}},
			}},
		}},
	})

	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("fourth nesting level must be a config error")
	}
	found := false
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Detail(), depthSectionMessage) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected truncation warning, got %v", resp.Diagnostics)
	}
}

func TestValidateConfig_AcceptsThreeLevels(t *testing.T) {
	r, cfg := schemaAndConfig(t, customComplianceFrameworkResourceModel{
		Name:               types.StringValue("three"),
		ForcedCloudVendors: types.SetNull(types.StringType),
		Sections: []sectionModel{{
			Name: types.StringValue("L1"),
			Sections: []midSectionModel{{
				Name: types.StringValue("L2"),
				Sections: []leafSectionModel{{
					Name: types.StringValue("L3"),
					Tests: []testModel{
						{RuleID: types.StringValue("r1"), RuleIDInFramework: types.StringValue("1.1.1")},
					},
				}},
			}},
		}},
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
