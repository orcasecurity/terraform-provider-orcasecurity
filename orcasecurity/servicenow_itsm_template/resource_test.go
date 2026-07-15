package servicenow_itsm_template

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// The ITSM template resource must advertise its own type-name suffix so it does not collide with
// the SIR template variant.
func TestNewServiceNowITSMTemplateResource_Metadata(t *testing.T) {
	r := NewServiceNowITSMTemplateResource()
	if r == nil {
		t.Fatal("constructor returned nil")
	}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_integration_servicenow_itsm_template" {
		t.Errorf("TypeName: got %q", resp.TypeName)
	}
}

// The resource must build a schema carrying the ServiceNow-specific mapping attribute and the
// shared template_name key — confirming the common Spec is wired through.
func TestNewServiceNowITSMTemplateResource_Schema(t *testing.T) {
	r := NewServiceNowITSMTemplateResource().(resource.ResourceWithConfigure)
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diags: %v", resp.Diagnostics)
	}
	for _, name := range []string{"template_name", "mapping_json", "resolution_status", "resource_id"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("schema missing attribute %q", name)
		}
	}
}

// ITSM templates do not accept business units, so the attribute must be absent from the schema.
func TestNewServiceNowITSMTemplateResource_NoBusinessUnits(t *testing.T) {
	r := NewServiceNowITSMTemplateResource().(resource.ResourceWithConfigure)
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if _, ok := resp.Schema.Attributes["business_units"]; ok {
		t.Errorf("ITSM template schema must not expose business_units")
	}
}

// The constructor must produce an import-capable resource so `terraform import` works by
// template_name.
func TestNewServiceNowITSMTemplateResource_ImplementsImport(t *testing.T) {
	r := NewServiceNowITSMTemplateResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatal("resource must implement ResourceWithImportState")
	}
}
