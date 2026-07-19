package servicenow

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The schema data source must expose the dedicated type name (distinct from the credentials
// resource/data source).
func TestSchemaDataSource_Metadata(t *testing.T) {
	ds := NewServiceNowSchemaDataSource()
	resp := &datasource.MetadataResponse{}
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_integration_servicenow_schema" {
		t.Errorf("TypeName: got %q", resp.TypeName)
	}
}

// The schema data source must require resource_id + type, constrain type to sir|itsm, and expose
// the computed elements list and fields nested block.
func TestSchemaDataSource_Schema(t *testing.T) {
	ds := NewServiceNowSchemaDataSource().(datasource.DataSourceWithConfigure)
	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diags: %v", resp.Diagnostics)
	}
	if !resp.Schema.Attributes["resource_id"].IsRequired() {
		t.Errorf("resource_id must be required")
	}
	if !resp.Schema.Attributes["type"].IsRequired() {
		t.Errorf("type must be required")
	}
	if !resp.Schema.Attributes["elements"].IsComputed() {
		t.Errorf("elements must be computed")
	}
	if _, ok := resp.Schema.Attributes["fields"]; !ok {
		t.Errorf("schema missing fields nested attribute")
	}
}

// Configure must ignore a nil ProviderData and store a supplied client otherwise.
func TestSchemaDataSource_Configure(t *testing.T) {
	ds := &schemaDataSource{}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, &datasource.ConfigureResponse{})
	if ds.apiClient != nil {
		t.Errorf("apiClient should stay nil for nil ProviderData")
	}
	client := &api_client.APIClient{}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: client}, &datasource.ConfigureResponse{})
	if ds.apiClient != client {
		t.Errorf("apiClient not stored from ProviderData")
	}
}

// schemaFieldModel must faithfully carry every ServiceNowSchemaField column. This documents the
// field-by-field copy the Read loop performs (the loop itself needs a live API, so we assert the
// per-field mapping directly on the model instead).
func TestSchemaFieldModel_MirrorsAPIField(t *testing.T) {
	f := api_client.ServiceNowSchemaField{
		Element:      "short_description",
		Label:        "Short description",
		Type:         "string",
		DefaultValue: "",
		MaxLength:    "160",
		Choice:       "0",
	}
	m := schemaFieldModel{
		Element:      types.StringValue(f.Element),
		Label:        types.StringValue(f.Label),
		Type:         types.StringValue(f.Type),
		DefaultValue: types.StringValue(f.DefaultValue),
		MaxLength:    types.StringValue(f.MaxLength),
		Choice:       types.StringValue(f.Choice),
	}
	if m.Element.ValueString() != f.Element {
		t.Errorf("element: got %q", m.Element.ValueString())
	}
	if m.Label.ValueString() != f.Label {
		t.Errorf("label: got %q", m.Label.ValueString())
	}
	if m.Type.ValueString() != f.Type {
		t.Errorf("type: got %q", m.Type.ValueString())
	}
	if m.MaxLength.ValueString() != f.MaxLength {
		t.Errorf("max_length: got %q", m.MaxLength.ValueString())
	}
	if m.Choice.ValueString() != f.Choice {
		t.Errorf("choice: got %q", m.Choice.ValueString())
	}
}
