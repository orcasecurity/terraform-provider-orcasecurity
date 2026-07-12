package monday_template

import (
	"context"
	"encoding/json"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// decodeMappings must copy each JSON-string state field into the matching config RawMessage.
func TestDecodeMappings_PopulatesConfigFromState(t *testing.T) {
	s := &state{
		MappingJSON:             common.NewOrcaMappingValue(`{"status_14":{"value":"0"}}`),
		AlertStatusMappingJSON:  jsontypes.NewNormalizedValue(`{"snoozed":"1"}`),
		TicketStatusMappingJSON: jsontypes.NewNormalizedValue(`{"2":{"status":"dismissed"}}`),
	}
	var cfg api_client.MondayTemplateConfig
	var diags diag.Diagnostics
	decodeMappings(s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if string(cfg.Mapping) != `{"status_14":{"value":"0"}}` {
		t.Errorf("mapping mismatch: %s", cfg.Mapping)
	}
	if string(cfg.AlertStatusMapping) != `{"snoozed":"1"}` {
		t.Errorf("alert_status_mapping mismatch: %s", cfg.AlertStatusMapping)
	}
	if string(cfg.TicketStatusMapping) != `{"2":{"status":"dismissed"}}` {
		t.Errorf("ticket_status_mapping mismatch: %s", cfg.TicketStatusMapping)
	}
}

// Invalid JSON in a mapping field must surface a plan-time diagnostic, not silently pass.
func TestDecodeMappings_InvalidJSONSurfacesError(t *testing.T) {
	s := &state{MappingJSON: common.NewOrcaMappingValue(`{not json`)}
	var cfg api_client.MondayTemplateConfig
	var diags diag.Diagnostics
	decodeMappings(s, &cfg, &diags)
	if !diags.HasError() {
		t.Fatal("expected error diag for invalid JSON")
	}
}

// The API value is stored verbatim; whitespace differences are absorbed by the mapping type's
// semantic equality (not by re-marshalling here), so the plan stays stable.
func TestEncodeMappings_StoresApiValueSemanticallyEqual(t *testing.T) {
	planned := common.NewOrcaMappingValue(`{"status_14":{"value":"0"}}`)
	s := &state{MappingJSON: planned}
	cfg := api_client.MondayTemplateConfig{
		Mapping: json.RawMessage(`{ "status_14": { "value": "0" } }`),
	}
	var diags diag.Diagnostics
	encodeMappings(s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	eq, d := s.MappingJSON.StringSemanticEquals(context.Background(), planned)
	if d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if !eq {
		t.Errorf("stored API value should be semantically equal to the plan, got %s", s.MappingJSON.ValueString())
	}
}

// An empty/absent API mapping must leave a null planned value null (no spurious "" diff).
func TestEncodeMappings_EmptyAPIKeepsNullPlan(t *testing.T) {
	s := &state{
		MappingJSON:             common.NewOrcaMappingNull(),
		AlertStatusMappingJSON:  jsontypes.NewNormalizedNull(),
		TicketStatusMappingJSON: jsontypes.NewNormalizedNull(),
	}
	var cfg api_client.MondayTemplateConfig // all mapping fields nil
	var diags diag.Diagnostics
	encodeMappings(s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !s.MappingJSON.IsNull() {
		t.Errorf("expected null mapping, got %v", s.MappingJSON)
	}
}

// The friendly HCL form (bare-string shorthand + literal objects) must round-trip through
// decode -> API -> encode semantically unchanged, even though state ends up holding the API's
// expanded {"orca":...} wire form.
func TestDecodeEncodeRoundTrip(t *testing.T) {
	orig := common.NewOrcaMappingValue(`{"long_text_mkn8v2sp":["alert_id"],"person":[{"value":{"id":"66396150","kind":"person"}}]}`)
	s := &state{MappingJSON: orig}
	var cfg api_client.MondayTemplateConfig
	var diags diag.Diagnostics
	decodeMappings(s, &cfg, &diags)
	// Payload carries the expanded wire form the API expects.
	if string(cfg.Mapping) != `{"long_text_mkn8v2sp":[{"orca":"alert_id"}],"person":[{"value":{"id":"66396150","kind":"person"}}]}` {
		t.Errorf("payload not expanded to wire form: %s", cfg.Mapping)
	}
	encodeMappings(s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	eq, d := s.MappingJSON.StringSemanticEquals(context.Background(), orig)
	if d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if !eq {
		t.Errorf("round-trip drifted: got %s", s.MappingJSON.ValueString())
	}
}
