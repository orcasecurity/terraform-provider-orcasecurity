package akamai

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BuildPayload must copy template/enabled/default plus all three EdgeGrid secrets and the host
// into the API envelope. Akamai does not support business_units.
func TestBuildPayload_PopulatesAllFields(t *testing.T) {
	s := &state{
		AccessToken:  types.StringValue("acc-tok"),
		ClientToken:  types.StringValue("cli-tok"),
		ClientSecret: types.StringValue("cli-sec"),
		Host:         types.StringValue("akab-x.luna.akamaiapis.net"),
	}
	s.TemplateName = types.StringValue("tf-acc-test-akamai")
	s.IsEnabled = types.BoolValue(true)
	s.IsDefault = types.BoolValue(false)

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.TemplateName != "tf-acc-test-akamai" {
		t.Errorf("template_name mismatch: %q", got.TemplateName)
	}
	if !got.IsEnabled || got.IsDefault {
		t.Errorf("enabled/default mismatch: enabled=%v default=%v", got.IsEnabled, got.IsDefault)
	}
	if got.Config.AccessToken != "acc-tok" || got.Config.ClientToken != "cli-tok" || got.Config.ClientSecret != "cli-sec" {
		t.Errorf("credentials mismatch: %+v", got.Config)
	}
	if got.Config.Host != "akab-x.luna.akamaiapis.net" {
		t.Errorf("host mismatch: %q", got.Config.Host)
	}
	if got.BusinessUnits != nil {
		t.Errorf("Akamai must not forward business_units, got %v", got.BusinessUnits)
	}
}

// Extract must map the computed envelope fields and echo the host the API returns back into
// state (host is a Required, non-Computed attr but the API round-trips it).
func TestExtract_MapsComputedFieldsAndEchoesHost(t *testing.T) {
	o := &api_client.AkamaiExternalServiceConfig{
		ID:           "uuid-123",
		TemplateName: "tf-acc-test-akamai",
		IsEnabled:    true,
		IsDefault:    true,
		Config:       api_client.AkamaiConfig{Host: "akab-y.luna.akamaiapis.net"},
	}
	s := &state{Host: types.StringValue("akab-x.luna.akamaiapis.net")}
	var diags diag.Diagnostics
	got := extract(o, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.ID != "uuid-123" || got.TemplateName != "tf-acc-test-akamai" {
		t.Errorf("id/template mismatch: %+v", got)
	}
	if !got.IsEnabled || !got.IsDefault {
		t.Errorf("enabled/default mismatch: %+v", got)
	}
	if s.Host.ValueString() != "akab-y.luna.akamaiapis.net" {
		t.Errorf("Extract must echo the API host into state, got %q", s.Host.ValueString())
	}
}

// When the API returns an empty host, Extract must leave the planned host untouched so no
// spurious diff appears (host is never blanked out from a bad response).
func TestExtract_EmptyHostKeepsPlanned(t *testing.T) {
	o := &api_client.AkamaiExternalServiceConfig{ID: "uuid", TemplateName: "t", Config: api_client.AkamaiConfig{Host: ""}}
	s := &state{Host: types.StringValue("akab-x.luna.akamaiapis.net")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.Host.ValueString() != "akab-x.luna.akamaiapis.net" {
		t.Errorf("empty API host must not clobber planned host, got %q", s.Host.ValueString())
	}
}

// Extract must not overwrite the planned EdgeGrid secrets (the API never returns them).
func TestExtract_DoesNotTouchSecrets(t *testing.T) {
	o := &api_client.AkamaiExternalServiceConfig{ID: "uuid", TemplateName: "t"}
	s := &state{
		AccessToken:  types.StringValue("planned-acc"),
		ClientToken:  types.StringValue("planned-cli"),
		ClientSecret: types.StringValue("planned-sec"),
	}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.AccessToken.ValueString() != "planned-acc" || s.ClientToken.ValueString() != "planned-cli" || s.ClientSecret.ValueString() != "planned-sec" {
		t.Errorf("Extract must not overwrite secrets, got %q/%q/%q",
			s.AccessToken.ValueString(), s.ClientToken.ValueString(), s.ClientSecret.ValueString())
	}
}

var _ cc.State = &state{}
