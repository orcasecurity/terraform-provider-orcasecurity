package pagerduty

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BuildPayload must copy the template/enabled/default fields and the sensitive integration key
// into the API envelope. PagerDuty does not support business_units, so none is forwarded.
func TestBuildPayload_PopulatesAllFields(t *testing.T) {
	s := &state{IntegrationKey: types.StringValue("int-key")}
	s.TemplateName = types.StringValue("tf-acc-test-pd")
	s.IsEnabled = types.BoolValue(true)
	s.IsDefault = types.BoolValue(true)

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.TemplateName != "tf-acc-test-pd" {
		t.Errorf("template_name mismatch: %q", got.TemplateName)
	}
	if !got.IsEnabled || !got.IsDefault {
		t.Errorf("enabled/default mismatch: enabled=%v default=%v", got.IsEnabled, got.IsDefault)
	}
	if got.Config.IntegrationKey != "int-key" {
		t.Errorf("integration_key mismatch: %q", got.Config.IntegrationKey)
	}
	if got.BusinessUnits != nil {
		t.Errorf("PagerDuty must not forward business_units, got %v", got.BusinessUnits)
	}
}

// An empty integration key must serialize to an empty Config.IntegrationKey so the update path's
// `omitempty` drops it and the API keeps the secret already in SSM.
func TestBuildPayload_EmptyKeyProducesEmptyString(t *testing.T) {
	s := &state{IntegrationKey: types.StringNull()}
	s.TemplateName = types.StringValue("tf-acc-test-pd")

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.Config.IntegrationKey != "" {
		t.Errorf("expected empty integration_key, got %q", got.Config.IntegrationKey)
	}
}

// Extract must map the API envelope's computed fields onto the APIObject.
func TestExtract_MapsComputedFields(t *testing.T) {
	o := &api_client.PagerDutyExternalServiceConfig{
		ID:           "uuid-456",
		TemplateName: "tf-acc-test-pd",
		IsEnabled:    true,
		IsDefault:    false,
	}
	s := &state{}
	var diags diag.Diagnostics
	got := extract(o, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.ID != "uuid-456" || got.TemplateName != "tf-acc-test-pd" {
		t.Errorf("id/template mismatch: %+v", got)
	}
	if !got.IsEnabled || got.IsDefault {
		t.Errorf("enabled/default mismatch: %+v", got)
	}
	// PagerDuty is a non-BU variant: Extract never carries business_units.
	if got.BusinessUnits != nil {
		t.Errorf("expected nil business_units, got %v", got.BusinessUnits)
	}
}

// Extract must not overwrite the planned sensitive integration key (API never returns it).
func TestExtract_DoesNotTouchSensitiveKey(t *testing.T) {
	o := &api_client.PagerDutyExternalServiceConfig{ID: "uuid", TemplateName: "t"}
	s := &state{IntegrationKey: types.StringValue("planned-key")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.IntegrationKey.ValueString() != "planned-key" {
		t.Errorf("Extract must not overwrite integration_key, got %q", s.IntegrationKey.ValueString())
	}
}

var _ cc.State = &state{}
