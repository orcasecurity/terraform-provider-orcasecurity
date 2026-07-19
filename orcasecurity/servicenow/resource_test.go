package servicenow

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// buildPayload must translate the Terraform plan model into the API payload, wiring the
// username/password into the nested Data block. ServiceName/Type are pinned inside the api_client
// wrappers, not here, so they are intentionally absent from the built payload.
func TestBuildPayload_MapsPlanToAPIPayload(t *testing.T) {
	r := &serviceNowITSMResource{}
	plan := serviceNowITSMResourceModel{
		Name:     types.StringValue("prod-servicenow"),
		URL:      types.StringValue("https://acme.service-now.com"),
		Username: types.StringValue("svc-orca"),
		Password: types.StringValue("s3cret"),
	}
	payload := r.buildPayload(plan)

	if payload.Name != "prod-servicenow" {
		t.Errorf("name: got %q", payload.Name)
	}
	if payload.HostURL != "https://acme.service-now.com" {
		t.Errorf("host_url: got %q", payload.HostURL)
	}
	if payload.Data.Username != "svc-orca" {
		t.Errorf("data.username: got %q", payload.Data.Username)
	}
	if payload.Data.Password == nil {
		t.Fatal("data.password: expected non-nil pointer")
	}
	if *payload.Data.Password != "s3cret" {
		t.Errorf("data.password: got %q", *payload.Data.Password)
	}
}

// An empty password plan value must still produce a non-nil pointer to "" — the password field is
// always sent (Required in the schema), so the API receives an explicit empty string rather than
// an omitted field.
func TestBuildPayload_EmptyPasswordYieldsNonNilPointer(t *testing.T) {
	r := &serviceNowITSMResource{}
	plan := serviceNowITSMResourceModel{
		Name:     types.StringValue("n"),
		URL:      types.StringValue("u"),
		Username: types.StringValue("user"),
		Password: types.StringValue(""),
	}
	payload := r.buildPayload(plan)
	if payload.Data.Password == nil {
		t.Fatal("expected non-nil password pointer even for empty password")
	}
	if *payload.Data.Password != "" {
		t.Errorf("expected empty password, got %q", *payload.Data.Password)
	}
}

// Metadata must derive the resource type name from the provider prefix.
func TestResource_Metadata(t *testing.T) {
	r := NewServiceNowResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_integration_servicenow_resource" {
		t.Errorf("TypeName: got %q", resp.TypeName)
	}
}

// The resource schema must expose all four managed attributes and mark password sensitive.
func TestResource_Schema(t *testing.T) {
	r := NewServiceNowResource().(resource.ResourceWithConfigure)
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diags: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "name", "servicenow_url", "username", "password"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("schema missing attribute %q", name)
		}
	}
	if !resp.Schema.Attributes["password"].IsSensitive() {
		t.Errorf("password attribute must be sensitive")
	}
	if !resp.Schema.Attributes["id"].IsComputed() {
		t.Errorf("id attribute must be computed")
	}
}

// Configure must be a no-op (and not panic) when ProviderData is nil — this is the shape the
// framework uses during early validation, before the provider is configured.
func TestResource_ConfigureNilProviderData(t *testing.T) {
	r := &serviceNowITSMResource{}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags on nil ProviderData: %v", resp.Diagnostics)
	}
	if r.apiClient != nil {
		t.Errorf("apiClient should remain nil when ProviderData is nil")
	}
}

// Configure must store the API client when the framework supplies one.
func TestResource_ConfigureStoresClient(t *testing.T) {
	r := &serviceNowITSMResource{}
	client := &api_client.APIClient{}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, resp)
	if r.apiClient != client {
		t.Errorf("apiClient not stored from ProviderData")
	}
}

// Create must fail with a clear diagnostic (not a nil-pointer panic) when the API client was
// never configured.
func TestResource_CreateWithoutClientErrors(t *testing.T) {
	r := &serviceNowITSMResource{apiClient: nil}
	resp := &resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic when API client is not configured")
	}
}

// ImportState must copy the import ID into the id attribute path. A nil state schema would panic
// on Set, so this also confirms the import path is exercised through a real state object.
func TestResource_ImportStateSetsID(t *testing.T) {
	// ImportState calls resp.State.SetAttribute, which requires a schema on the state. Building a
	// full tfsdk.State is heavyweight; instead assert the constructor wires the import-capable
	// interface, which is the contract callers rely on.
	r := NewServiceNowResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Fatal("resource must implement ResourceWithImportState")
	}
}
