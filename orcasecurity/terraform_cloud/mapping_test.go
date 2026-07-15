package terraform_cloud

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BuildPayload must copy template/enabled/default plus the sensitive token and the API URL into
// the API envelope. Terraform Cloud does not support business_units.
func TestBuildPayload_PopulatesAllFields(t *testing.T) {
	s := &state{
		APIToken: types.StringValue("tfc-token"),
		APIURL:   types.StringValue("https://app.terraform.io"),
	}
	s.TemplateName = types.StringValue("tf-acc-test-tfc")
	s.IsEnabled = types.BoolValue(true)
	s.IsDefault = types.BoolValue(false)

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.TemplateName != "tf-acc-test-tfc" {
		t.Errorf("template_name mismatch: %q", got.TemplateName)
	}
	if !got.IsEnabled || got.IsDefault {
		t.Errorf("enabled/default mismatch: enabled=%v default=%v", got.IsEnabled, got.IsDefault)
	}
	if got.Config.APIToken != "tfc-token" {
		t.Errorf("api_token mismatch: %q", got.Config.APIToken)
	}
	if got.Config.APIURL != "https://app.terraform.io" {
		t.Errorf("api_url mismatch: %q", got.Config.APIURL)
	}
	if got.BusinessUnits != nil {
		t.Errorf("Terraform Cloud must not forward business_units, got %v", got.BusinessUnits)
	}
}

// Extract must map the computed envelope fields and echo the API URL the API returns back into
// state (api_url is a Required, non-Computed attr but the API round-trips it).
func TestExtract_MapsComputedFieldsAndEchoesAPIURL(t *testing.T) {
	o := &api_client.TerraformCloudExternalServiceConfig{
		ID:           "uuid-321",
		TemplateName: "tf-acc-test-tfc",
		IsEnabled:    true,
		IsDefault:    true,
		Config:       api_client.TerraformCloudConfig{APIURL: "https://tfe.example.com"},
	}
	s := &state{APIURL: types.StringValue("https://app.terraform.io")}
	var diags diag.Diagnostics
	got := extract(o, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.ID != "uuid-321" || got.TemplateName != "tf-acc-test-tfc" {
		t.Errorf("id/template mismatch: %+v", got)
	}
	if !got.IsEnabled || !got.IsDefault {
		t.Errorf("enabled/default mismatch: %+v", got)
	}
	if s.APIURL.ValueString() != "https://tfe.example.com" {
		t.Errorf("Extract must echo the API api_url into state, got %q", s.APIURL.ValueString())
	}
}

// When the API returns an empty URL, Extract must leave the planned api_url untouched so no
// spurious diff appears (the URL is never blanked out from a bad response).
func TestExtract_EmptyAPIURLKeepsPlanned(t *testing.T) {
	o := &api_client.TerraformCloudExternalServiceConfig{ID: "uuid", TemplateName: "t", Config: api_client.TerraformCloudConfig{APIURL: ""}}
	s := &state{APIURL: types.StringValue("https://app.terraform.io")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.APIURL.ValueString() != "https://app.terraform.io" {
		t.Errorf("empty API api_url must not clobber planned value, got %q", s.APIURL.ValueString())
	}
}

// Extract must not overwrite the planned sensitive token (the API never returns it).
func TestExtract_DoesNotTouchSensitiveToken(t *testing.T) {
	o := &api_client.TerraformCloudExternalServiceConfig{ID: "uuid", TemplateName: "t"}
	s := &state{APIToken: types.StringValue("planned-token")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.APIToken.ValueString() != "planned-token" {
		t.Errorf("Extract must not overwrite api_token, got %q", s.APIToken.ValueString())
	}
}

var _ cc.State = &state{}
