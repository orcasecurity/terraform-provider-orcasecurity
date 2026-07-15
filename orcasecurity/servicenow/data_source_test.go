package servicenow

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// The name-lookup data source must derive its type name from the provider prefix. It shares the
// same type name as the managed resource (both back the sn_incidents credentials resource).
func TestDataSource_Metadata(t *testing.T) {
	ds := NewServiceNowDataSource()
	resp := &datasource.MetadataResponse{}
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_integration_servicenow_resource" {
		t.Errorf("TypeName: got %q", resp.TypeName)
	}
}

// The lookup data source schema must require name and expose the computed id/url/username.
func TestDataSource_Schema(t *testing.T) {
	ds := NewServiceNowDataSource().(datasource.DataSourceWithConfigure)
	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diags: %v", resp.Diagnostics)
	}
	if !resp.Schema.Attributes["name"].IsRequired() {
		t.Errorf("name must be required")
	}
	for _, name := range []string{"id", "servicenow_url", "username"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("schema missing attribute %q", name)
			continue
		}
		if !attr.IsComputed() {
			t.Errorf("attribute %q must be computed", name)
		}
	}
}

// Configure must ignore a nil ProviderData and store a supplied client otherwise.
func TestDataSource_Configure(t *testing.T) {
	ds := &itsmDataSource{}
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
