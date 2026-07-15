package splunk

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BuildPayload must copy template/enabled/default, the HEC URL, the sensitive token, and the
// self-signed-cert flag into the API envelope. Splunk does not support business_units.
func TestBuildPayload_PopulatesAllFields(t *testing.T) {
	s := &state{
		URL:                 types.StringValue("https://splunk.example.com:8088/services/collector/event"),
		Token:               types.StringValue("hec-token"),
		AllowSelfSignedCert: types.BoolValue(true),
	}
	s.TemplateName = types.StringValue("tf-acc-test-splunk")
	s.IsEnabled = types.BoolValue(true)
	s.IsDefault = types.BoolValue(false)

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.TemplateName != "tf-acc-test-splunk" {
		t.Errorf("template_name mismatch: %q", got.TemplateName)
	}
	if got.Config.URL != "https://splunk.example.com:8088/services/collector/event" {
		t.Errorf("url mismatch: %q", got.Config.URL)
	}
	if got.Config.Token != "hec-token" {
		t.Errorf("token mismatch: %q", got.Config.Token)
	}
	if !got.Config.AllowSelfSignedCert {
		t.Errorf("allow_self_signed_cert mismatch: %v", got.Config.AllowSelfSignedCert)
	}
	if got.BusinessUnits != nil {
		t.Errorf("Splunk must not forward business_units, got %v", got.BusinessUnits)
	}
}

// The self-signed-cert flag defaults to false and must serialize as an explicit false (the API
// field has no omitempty; a missing value would be ambiguous).
func TestBuildPayload_SelfSignedCertFalse(t *testing.T) {
	s := &state{
		URL:                 types.StringValue("https://x"),
		Token:               types.StringValue("t"),
		AllowSelfSignedCert: types.BoolValue(false),
	}
	s.TemplateName = types.StringValue("tf-acc-test-splunk")

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.Config.AllowSelfSignedCert {
		t.Errorf("expected allow_self_signed_cert false, got true")
	}
}

// Extract must map the computed envelope fields, echo the URL, and round-trip the self-signed
// cert flag from the API response into state.
func TestExtract_MapsComputedFieldsEchoesURLAndCert(t *testing.T) {
	o := &api_client.SplunkExternalServiceConfig{
		ID:           "uuid-splunk",
		TemplateName: "tf-acc-test-splunk",
		IsEnabled:    true,
		IsDefault:    false,
		Config: api_client.SplunkConfig{
			URL:                 "https://returned.example.com:8088/services/collector/event",
			AllowSelfSignedCert: true,
		},
	}
	s := &state{URL: types.StringValue("https://planned")}
	var diags diag.Diagnostics
	got := extract(o, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.ID != "uuid-splunk" || got.TemplateName != "tf-acc-test-splunk" {
		t.Errorf("id/template mismatch: %+v", got)
	}
	if s.URL.ValueString() != "https://returned.example.com:8088/services/collector/event" {
		t.Errorf("Extract must echo the API URL, got %q", s.URL.ValueString())
	}
	if !s.AllowSelfSignedCert.ValueBool() {
		t.Errorf("Extract must round-trip allow_self_signed_cert, got %v", s.AllowSelfSignedCert.ValueBool())
	}
}

// When the API returns an empty URL, Extract must keep the planned URL (never blank it out).
func TestExtract_EmptyURLKeepsPlanned(t *testing.T) {
	o := &api_client.SplunkExternalServiceConfig{ID: "uuid", TemplateName: "t", Config: api_client.SplunkConfig{URL: ""}}
	s := &state{URL: types.StringValue("https://planned")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.URL.ValueString() != "https://planned" {
		t.Errorf("empty API URL must not clobber planned URL, got %q", s.URL.ValueString())
	}
}

// Extract must not overwrite the planned sensitive HEC token (API never returns it).
func TestExtract_DoesNotTouchSensitiveToken(t *testing.T) {
	o := &api_client.SplunkExternalServiceConfig{ID: "uuid", TemplateName: "t"}
	s := &state{Token: types.StringValue("planned-token")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.Token.ValueString() != "planned-token" {
		t.Errorf("Extract must not overwrite token, got %q", s.Token.ValueString())
	}
}

var _ cc.State = &state{}
