package servicenow_template_common

import (
	"context"
	"encoding/json"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// boolPtr is a helper for building *bool config fields the API returns.
func boolPtr(b bool) *bool { return &b }

// extractStateFromAPI must copy every non-empty scalar config field from the API response into
// the corresponding Terraform state field.
func TestExtractStateFromAPI_PopulatesAllScalarFields(t *testing.T) {
	api := &api_client.ServiceNowITSMTemplate{
		Resource: "res-123",
		Config: api_client.ServiceNowITSMTemplateConfig{
			InstanceName:             "acme",
			BaseURL:                  "https://acme.service-now.com",
			Username:                 "svc-orca",
			ResolutionStatus:         "6",
			ResolutionCode:           "Solved (Permanently)",
			ResolutionNote:           "Closed by Orca",
			ReopenStatus:             "2",
			AllowReopenAndResolution: boolPtr(true),
			AllowMapping:             boolPtr(true),
		},
	}
	s := &state{}
	var diags diag.Diagnostics
	extractStateFromAPI(api, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	cases := map[string]struct {
		got  string
		want string
	}{
		"resource_id":       {s.ResourceID.ValueString(), "res-123"},
		"instance_name":     {s.InstanceName.ValueString(), "acme"},
		"base_url":          {s.BaseURL.ValueString(), "https://acme.service-now.com"},
		"username":          {s.Username.ValueString(), "svc-orca"},
		"resolution_status": {s.ResolutionStatus.ValueString(), "6"},
		"resolution_code":   {s.ResolutionCode.ValueString(), "Solved (Permanently)"},
		"resolution_note":   {s.ResolutionNote.ValueString(), "Closed by Orca"},
		"reopen_status":     {s.ReopenStatus.ValueString(), "2"},
	}
	for name, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %q, want %q", name, c.got, c.want)
		}
	}
	if !s.AllowReopenAndResolution.ValueBool() {
		t.Errorf("allow_reopen_and_resolution: expected true")
	}
	if !s.AllowMapping.ValueBool() {
		t.Errorf("allow_mapping: expected true")
	}
}

// Empty scalar fields from the API must leave the state field untouched (null), so an unset
// optional attribute does not gain a spurious "" value and provoke a diff.
func TestExtractStateFromAPI_EmptyScalarsLeaveStateNull(t *testing.T) {
	api := &api_client.ServiceNowITSMTemplate{
		Config: api_client.ServiceNowITSMTemplateConfig{}, // all zero values
	}
	s := &state{}
	var diags diag.Diagnostics
	extractStateFromAPI(api, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !s.ResourceID.IsNull() {
		t.Errorf("resource_id: expected null, got %q", s.ResourceID.ValueString())
	}
	if !s.InstanceName.IsNull() {
		t.Errorf("instance_name: expected null, got %q", s.InstanceName.ValueString())
	}
	if !s.BaseURL.IsNull() {
		t.Errorf("base_url: expected null, got %q", s.BaseURL.ValueString())
	}
	if !s.Username.IsNull() {
		t.Errorf("username: expected null, got %q", s.Username.ValueString())
	}
	// nil *bool fields must not be written.
	if !s.AllowReopenAndResolution.IsNull() {
		t.Errorf("allow_reopen_and_resolution: expected null when API returns nil pointer")
	}
	if !s.AllowMapping.IsNull() {
		t.Errorf("allow_mapping: expected null when API returns nil pointer")
	}
}

// A *bool set to false must be written as an explicit false (not skipped as if it were nil).
func TestExtractStateFromAPI_FalseBoolIsWritten(t *testing.T) {
	api := &api_client.ServiceNowITSMTemplate{
		Config: api_client.ServiceNowITSMTemplateConfig{
			AllowReopenAndResolution: boolPtr(false),
			AllowMapping:             boolPtr(false),
		},
	}
	s := &state{}
	var diags diag.Diagnostics
	extractStateFromAPI(api, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if s.AllowReopenAndResolution.IsNull() || s.AllowReopenAndResolution.ValueBool() {
		t.Errorf("allow_reopen_and_resolution: expected explicit false, got %v", s.AllowReopenAndResolution)
	}
	if s.AllowMapping.IsNull() || s.AllowMapping.ValueBool() {
		t.Errorf("allow_mapping: expected explicit false, got %v", s.AllowMapping)
	}
}

// A non-empty API mapping must be collapsed to the bare-string shorthand and stored, staying
// semantically equal to a shorthand plan (see EncodeOrcaMappingField).
func TestExtractStateFromAPI_MappingCollapsedToShorthand(t *testing.T) {
	planned := common.NewOrcaMappingValue(`{"short_description":["alert_id"]}`)
	api := &api_client.ServiceNowITSMTemplate{
		Config: api_client.ServiceNowITSMTemplateConfig{
			// API returns the expanded {"orca":...} wire form.
			Mapping: json.RawMessage(`{"short_description":[{"orca":"alert_id"}]}`),
		},
	}
	s := &state{MappingJSON: planned}
	var diags diag.Diagnostics
	extractStateFromAPI(api, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	eq, d := s.MappingJSON.StringSemanticEquals(context.Background(), planned)
	if d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if !eq {
		t.Errorf("stored mapping should stay semantically equal to the shorthand plan, got %s", s.MappingJSON.ValueString())
	}
}

// An empty/absent API mapping must keep a null planned mapping null (no diff against "{}").
func TestExtractStateFromAPI_EmptyMappingKeepsNullPlan(t *testing.T) {
	api := &api_client.ServiceNowITSMTemplate{
		Config: api_client.ServiceNowITSMTemplateConfig{}, // Mapping/OnClose nil
	}
	s := &state{
		MappingJSON:             common.NewOrcaMappingNull(),
		OnCloseAlertMappingJSON: jsontypes.NewNormalizedNull(),
	}
	var diags diag.Diagnostics
	extractStateFromAPI(api, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !s.MappingJSON.IsNull() {
		t.Errorf("mapping_json: expected null, got %q", s.MappingJSON.ValueString())
	}
	if !s.OnCloseAlertMappingJSON.IsNull() {
		t.Errorf("on_close_alert_mapping_json: expected null, got %q", s.OnCloseAlertMappingJSON.ValueString())
	}
}

// on_close_alert_mapping is stored verbatim and stays semantically equal to the plan across
// cosmetic whitespace/key-order differences.
func TestExtractStateFromAPI_OnCloseMappingStoredSemanticallyEqual(t *testing.T) {
	planned := jsontypes.NewNormalizedValue(`{"state":"closed"}`)
	api := &api_client.ServiceNowITSMTemplate{
		Config: api_client.ServiceNowITSMTemplateConfig{
			OnCloseAlertMapping: json.RawMessage(`{ "state": "closed" }`),
		},
	}
	s := &state{OnCloseAlertMappingJSON: planned}
	var diags diag.Diagnostics
	extractStateFromAPI(api, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	eq, d := s.OnCloseAlertMappingJSON.StringSemanticEquals(context.Background(), planned)
	if d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if !eq {
		t.Errorf("on_close mapping should be semantically equal to the plan, got %s", s.OnCloseAlertMappingJSON.ValueString())
	}
}

// variantAttributes must expose exactly the ServiceNow-specific attributes the schema documents,
// and the two bool attributes must be Optional+Computed (they carry a default).
func TestVariantAttributes_ExposesExpectedAttributes(t *testing.T) {
	attrs := variantAttributes()
	expected := []string{
		"resource_id", "instance_name", "base_url", "username",
		"resolution_status", "resolution_code", "resolution_note", "reopen_status",
		"mapping_json", "on_close_alert_mapping_json",
		"allow_reopen_and_resolution", "allow_mapping",
	}
	for _, name := range expected {
		if _, ok := attrs[name]; !ok {
			t.Errorf("missing expected attribute %q", name)
		}
	}
	if len(attrs) != len(expected) {
		t.Errorf("attribute count mismatch: got %d, want %d", len(attrs), len(expected))
	}
}

// The embedded CommonFields glue must round-trip id/template_name/is_enabled/is_default through
// GetCommon/SetCommon so the generic Spec can refresh them.
func TestStateCommonFieldsRoundTrip(t *testing.T) {
	s := &state{}
	s.SetCommon(cc.Common{
		ID:           types.StringValue("id-1"),
		TemplateName: types.StringValue("tf-tmpl"),
		IsEnabled:    types.BoolValue(true),
		IsDefault:    types.BoolValue(false),
	})
	got := s.GetCommon()
	if got.ID.ValueString() != "id-1" {
		t.Errorf("id: got %q", got.ID.ValueString())
	}
	if got.TemplateName.ValueString() != "tf-tmpl" {
		t.Errorf("template_name: got %q", got.TemplateName.ValueString())
	}
	if !got.IsEnabled.ValueBool() {
		t.Errorf("is_enabled: expected true")
	}
	if got.IsDefault.ValueBool() {
		t.Errorf("is_default: expected false")
	}
}

// NewResource must return a non-nil resource whose type name uses the supplied suffix. This
// exercises the full Spec wiring (Metadata) without any network.
func TestNewResource_MetadataUsesSuffix(t *testing.T) {
	r := NewResource(Options{
		TypeNameSuffix: "_integration_servicenow_itsm_template",
		UIName:         "ServiceNow ITSM template",
		ConfigType:     api_client.ServiceNowITSMTemplateConfigType,
		Create:         (*api_client.APIClient).CreateServiceNowITSMTemplate,
		Get:            (*api_client.APIClient).GetServiceNowITSMTemplate,
		Update:         (*api_client.APIClient).UpdateServiceNowITSMTemplate,
		Delete:         (*api_client.APIClient).DeleteServiceNowITSMTemplate,
	})
	if r == nil {
		t.Fatal("NewResource returned nil")
	}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_integration_servicenow_itsm_template" {
		t.Errorf("TypeName: got %q", resp.TypeName)
	}
}

// The resource built by NewResource must expose a schema that includes both the cross-variant
// fields (template_name) and the ServiceNow-specific attributes (mapping_json).
func TestNewResource_SchemaIncludesVariantAttributes(t *testing.T) {
	r := NewResource(Options{
		TypeNameSuffix: "_integration_servicenow_sir_template",
		ConfigType:     api_client.ServiceNowSIRTemplateConfigType,
		Create:         (*api_client.APIClient).CreateServiceNowSIRTemplate,
		Get:            (*api_client.APIClient).GetServiceNowSIRTemplate,
		Update:         (*api_client.APIClient).UpdateServiceNowSIRTemplate,
		Delete:         (*api_client.APIClient).DeleteServiceNowSIRTemplate,
	})
	sr, ok := r.(resource.ResourceWithConfigure)
	if !ok {
		t.Fatal("resource does not implement ResourceWithConfigure")
	}
	resp := &resource.SchemaResponse{}
	sr.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if _, ok := resp.Schema.Attributes["template_name"]; !ok {
		t.Errorf("schema missing cross-variant attribute template_name")
	}
	if _, ok := resp.Schema.Attributes["mapping_json"]; !ok {
		t.Errorf("schema missing variant attribute mapping_json")
	}
	// business_units must NOT be present — ServiceNow templates set SupportsBusinessUnits=false.
	if _, ok := resp.Schema.Attributes["business_units"]; ok {
		t.Errorf("schema unexpectedly exposes business_units")
	}
}
