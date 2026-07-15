package zscaler

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BuildPayload must copy template/enabled/default plus the OAuth secrets and the vanity domain
// into the API envelope. Zscaler does not support business_units.
func TestBuildPayload_PopulatesAllFields(t *testing.T) {
	s := &state{
		VanityDomain: types.StringValue("acme"),
		ClientID:     types.StringValue("client-id"),
		ClientSecret: types.StringValue("client-sec"),
	}
	s.TemplateName = types.StringValue("tf-acc-test-zscaler")
	s.IsEnabled = types.BoolValue(true)
	s.IsDefault = types.BoolValue(false)

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.TemplateName != "tf-acc-test-zscaler" {
		t.Errorf("template_name mismatch: %q", got.TemplateName)
	}
	if !got.IsEnabled || got.IsDefault {
		t.Errorf("enabled/default mismatch: enabled=%v default=%v", got.IsEnabled, got.IsDefault)
	}
	if got.Config.VanityDomain != "acme" {
		t.Errorf("vanity_domain mismatch: %q", got.Config.VanityDomain)
	}
	if got.Config.ClientID != "client-id" || got.Config.ClientSecret != "client-sec" {
		t.Errorf("credentials mismatch: %+v", got.Config)
	}
	if got.BusinessUnits != nil {
		t.Errorf("Zscaler must not forward business_units, got %v", got.BusinessUnits)
	}
}

// Extract must map the computed envelope fields and echo the vanity domain the API returns back
// into state (vanity_domain is a Required, non-Computed attr but the API round-trips it).
func TestExtract_MapsComputedFieldsAndEchoesVanityDomain(t *testing.T) {
	o := &api_client.ZscalerExternalServiceConfig{
		ID:           "uuid-456",
		TemplateName: "tf-acc-test-zscaler",
		IsEnabled:    true,
		IsDefault:    true,
		Config:       api_client.ZscalerConfig{VanityDomain: "acme-updated"},
	}
	s := &state{VanityDomain: types.StringValue("acme")}
	var diags diag.Diagnostics
	got := extract(o, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.ID != "uuid-456" || got.TemplateName != "tf-acc-test-zscaler" {
		t.Errorf("id/template mismatch: %+v", got)
	}
	if !got.IsEnabled || !got.IsDefault {
		t.Errorf("enabled/default mismatch: %+v", got)
	}
	if s.VanityDomain.ValueString() != "acme-updated" {
		t.Errorf("Extract must echo the API vanity_domain into state, got %q", s.VanityDomain.ValueString())
	}
}

// When the API returns an empty vanity domain, Extract must leave the planned value untouched so
// no spurious diff appears (the domain is never blanked out from a bad response).
func TestExtract_EmptyVanityDomainKeepsPlanned(t *testing.T) {
	o := &api_client.ZscalerExternalServiceConfig{ID: "uuid", TemplateName: "t", Config: api_client.ZscalerConfig{VanityDomain: ""}}
	s := &state{VanityDomain: types.StringValue("acme")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.VanityDomain.ValueString() != "acme" {
		t.Errorf("empty API vanity_domain must not clobber planned value, got %q", s.VanityDomain.ValueString())
	}
}

// Extract must not overwrite the planned OAuth secrets (the API never returns them).
func TestExtract_DoesNotTouchSecrets(t *testing.T) {
	o := &api_client.ZscalerExternalServiceConfig{ID: "uuid", TemplateName: "t"}
	s := &state{
		ClientID:     types.StringValue("planned-id"),
		ClientSecret: types.StringValue("planned-sec"),
	}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.ClientID.ValueString() != "planned-id" || s.ClientSecret.ValueString() != "planned-sec" {
		t.Errorf("Extract must not overwrite secrets, got %q/%q", s.ClientID.ValueString(), s.ClientSecret.ValueString())
	}
}

var _ cc.State = &state{}
