package cloudflare

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BuildPayload must copy template/enabled/default plus the sensitive token into the API
// envelope. Cloudflare does not support business_units.
func TestBuildPayload_PopulatesAllFields(t *testing.T) {
	s := &state{APIToken: types.StringValue("cf-token")}
	s.TemplateName = types.StringValue("tf-acc-test-cloudflare")
	s.IsEnabled = types.BoolValue(true)
	s.IsDefault = types.BoolValue(false)

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.TemplateName != "tf-acc-test-cloudflare" {
		t.Errorf("template_name mismatch: %q", got.TemplateName)
	}
	if !got.IsEnabled || got.IsDefault {
		t.Errorf("enabled/default mismatch: enabled=%v default=%v", got.IsEnabled, got.IsDefault)
	}
	if got.Config.APIToken != "cf-token" {
		t.Errorf("api_token mismatch: %q", got.Config.APIToken)
	}
	if got.BusinessUnits != nil {
		t.Errorf("Cloudflare must not forward business_units, got %v", got.BusinessUnits)
	}
}

// Extract must map the API envelope's computed fields back onto the Common-shape APIObject.
func TestExtract_MapsComputedFields(t *testing.T) {
	o := &api_client.CloudflareExternalServiceConfig{
		ID:           "uuid-654",
		TemplateName: "tf-acc-test-cloudflare",
		IsEnabled:    true,
		IsDefault:    true,
	}
	s := &state{}
	var diags diag.Diagnostics
	got := extract(o, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.ID != "uuid-654" || got.TemplateName != "tf-acc-test-cloudflare" {
		t.Errorf("id/template mismatch: %+v", got)
	}
	if !got.IsEnabled || !got.IsDefault {
		t.Errorf("enabled/default mismatch: %+v", got)
	}
}

// Extract must not overwrite the planned sensitive token (the API never returns it).
func TestExtract_DoesNotTouchSensitiveToken(t *testing.T) {
	o := &api_client.CloudflareExternalServiceConfig{ID: "uuid", TemplateName: "t"}
	s := &state{APIToken: types.StringValue("planned-token")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.APIToken.ValueString() != "planned-token" {
		t.Errorf("Extract must not overwrite api_token, got %q", s.APIToken.ValueString())
	}
}

var _ cc.State = &state{}
