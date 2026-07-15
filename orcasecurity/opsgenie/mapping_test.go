package opsgenie

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BuildPayload must copy every populated state field (including the sensitive key and business
// units) into the API envelope shape.
func TestBuildPayload_PopulatesAllFields(t *testing.T) {
	s := &state{
		OpsgenieKey: types.StringValue("secret-key"),
	}
	s.TemplateName = types.StringValue("tf-acc-test-og")
	s.IsEnabled = types.BoolValue(true)
	s.IsDefault = types.BoolValue(false)
	s.BusinessUnits = testutils.StringSet(t, "bu-1", "bu-2")

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.TemplateName != "tf-acc-test-og" {
		t.Errorf("template_name mismatch: %q", got.TemplateName)
	}
	if !got.IsEnabled || got.IsDefault {
		t.Errorf("enabled/default mismatch: enabled=%v default=%v", got.IsEnabled, got.IsDefault)
	}
	if got.Config.OpsgenieKey != "secret-key" {
		t.Errorf("opsgenie_key mismatch: %q", got.Config.OpsgenieKey)
	}
	if !testutils.SameElements(got.BusinessUnits, []string{"bu-1", "bu-2"}) {
		t.Errorf("business_units mismatch: %v", got.BusinessUnits)
	}
}

// A null business_units set must produce a nil slice so `omitempty` drops the field entirely
// (matches the UI: unset, not an empty array).
func TestBuildPayload_NullBusinessUnitsOmitted(t *testing.T) {
	s := &state{OpsgenieKey: types.StringValue("k")}
	s.TemplateName = types.StringValue("tf-acc-test-og")
	s.BusinessUnits = types.SetNull(types.StringType)

	var diags diag.Diagnostics
	got := buildPayload(context.Background(), s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.BusinessUnits != nil {
		t.Errorf("expected nil business_units, got %v", got.BusinessUnits)
	}
}

// Extract must map the API envelope's computed fields (id, enabled, default, business_units)
// back onto the Common-shape APIObject.
func TestExtract_MapsComputedFields(t *testing.T) {
	o := &api_client.OpsgenieExternalServiceConfig{
		ID:            "uuid-123",
		TemplateName:  "tf-acc-test-og",
		IsEnabled:     true,
		IsDefault:     true,
		BusinessUnits: []string{"bu-1"},
	}
	s := &state{}
	var diags diag.Diagnostics
	got := extract(o, s, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.ID != "uuid-123" || got.TemplateName != "tf-acc-test-og" {
		t.Errorf("id/template mismatch: %+v", got)
	}
	if !got.IsEnabled || !got.IsDefault {
		t.Errorf("enabled/default mismatch: %+v", got)
	}
	if !testutils.SameElements(got.BusinessUnits, []string{"bu-1"}) {
		t.Errorf("business_units mismatch: %v", got.BusinessUnits)
	}
}

// Extract must never surface the sensitive opsgenie_key (the API never returns it); the state's
// planned key is left untouched by Extract so the framework keeps the configured value.
func TestExtract_DoesNotTouchSensitiveKey(t *testing.T) {
	o := &api_client.OpsgenieExternalServiceConfig{ID: "uuid", TemplateName: "t"}
	s := &state{OpsgenieKey: types.StringValue("planned-secret")}
	var diags diag.Diagnostics
	extract(o, s, &diags)
	if s.OpsgenieKey.ValueString() != "planned-secret" {
		t.Errorf("Extract must not overwrite the sensitive key, got %q", s.OpsgenieKey.ValueString())
	}
}

// compile-time guard: state must satisfy the cc.State interface used by the skeleton.
var _ cc.State = &state{}
