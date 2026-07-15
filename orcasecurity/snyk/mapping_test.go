package snyk

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BuildPayload must copy template/enabled/default plus the sensitive token and the region code
// into the API envelope. Snyk does not support business_units.
func TestBuildPayload_PopulatesAllFields(t *testing.T) {
	s := &state{
		APIToken: types.StringValue("snyk-token"),
		Region:   types.StringValue("EU"),
	}
	s.TemplateName = types.StringValue("tf-acc-test-snyk")
	s.IsEnabled = types.BoolValue(true)
	s.IsDefault = types.BoolValue(false)

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.TemplateName != "tf-acc-test-snyk" {
		t.Errorf("template_name mismatch: %q", got.TemplateName)
	}
	if got.Config.APIToken != "snyk-token" {
		t.Errorf("api_token mismatch: %q", got.Config.APIToken)
	}
	if got.Config.Region != "EU" {
		t.Errorf("region mismatch: %q", got.Config.Region)
	}
	if got.BusinessUnits != nil {
		t.Errorf("Snyk must not forward business_units, got %v", got.BusinessUnits)
	}
}

// Extract must map the computed envelope fields and echo the region the API returns back into
// state (region is a Required, non-Computed attr but the API round-trips it).
func TestExtract_MapsComputedFieldsAndEchoesRegion(t *testing.T) {
	o := &api_client.SnykExternalServiceConfig{
		ID:           "uuid-789",
		TemplateName: "tf-acc-test-snyk",
		IsEnabled:    true,
		IsDefault:    true,
		Config:       api_client.SnykConfig{Region: "US2"},
	}
	s := &state{Region: types.StringValue("EU")}
	var diags diag.Diagnostics
	got := extract(o, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.ID != "uuid-789" || got.TemplateName != "tf-acc-test-snyk" {
		t.Errorf("id/template mismatch: %+v", got)
	}
	if !got.IsEnabled || !got.IsDefault {
		t.Errorf("enabled/default mismatch: %+v", got)
	}
	if s.Region.ValueString() != "US2" {
		t.Errorf("Extract must echo the API region into state, got %q", s.Region.ValueString())
	}
}

// When the API returns an empty region, Extract must leave the planned region untouched so no
// spurious diff appears (region is never blanked out from a bad response).
func TestExtract_EmptyRegionKeepsPlanned(t *testing.T) {
	o := &api_client.SnykExternalServiceConfig{ID: "uuid", TemplateName: "t", Config: api_client.SnykConfig{Region: ""}}
	s := &state{Region: types.StringValue("AU")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.Region.ValueString() != "AU" {
		t.Errorf("empty API region must not clobber planned region, got %q", s.Region.ValueString())
	}
}

// Extract must not overwrite the planned sensitive token (API never returns it).
func TestExtract_DoesNotTouchSensitiveToken(t *testing.T) {
	o := &api_client.SnykExternalServiceConfig{ID: "uuid", TemplateName: "t"}
	s := &state{APIToken: types.StringValue("planned-token")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.APIToken.ValueString() != "planned-token" {
		t.Errorf("Extract must not overwrite api_token, got %q", s.APIToken.ValueString())
	}
}

var _ cc.State = &state{}
