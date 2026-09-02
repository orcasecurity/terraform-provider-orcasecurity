package custom_compliance_framework

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

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

func TestSchema_LoosenedAndAdditive(t *testing.T) {
	r := &customComplianceFrameworkResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	attrs := schemaResp.Schema.Attributes

	desc, ok := attrs["description"].(schema.StringAttribute)
	if !ok || !desc.Optional || desc.Required {
		t.Errorf("description must be Optional, got %#v", attrs["description"])
	}
	if !strings.Contains(desc.Description, "null") {
		t.Errorf("description schema must mention JSON null, got %q", desc.Description)
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
