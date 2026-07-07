package monday_template

import (
	"encoding/json"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// decodeMappings must copy each JSON-string state field into the matching config RawMessage.
func TestDecodeMappings_PopulatesConfigFromState(t *testing.T) {
	s := &state{
		MappingJSON:             types.StringValue(`{"status_14":{"value":"0"}}`),
		AlertStatusMappingJSON:  types.StringValue(`{"snoozed":"1"}`),
		TicketStatusMappingJSON: types.StringValue(`{"2":{"status":"dismissed"}}`),
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
	s := &state{MappingJSON: types.StringValue(`{not json`)}
	var cfg api_client.MondayTemplateConfig
	var diags diag.Diagnostics
	decodeMappings(s, &cfg, &diags)
	if !diags.HasError() {
		t.Fatal("expected error diag for invalid JSON")
	}
}

// API responses with cosmetic whitespace must normalize to compact JSON so plans don't drift.
func TestEncodeMappings_NormalizesWhitespaceFromAPI(t *testing.T) {
	s := &state{MappingJSON: types.StringValue(`{"status_14":{"value":"0"}}`)}
	cfg := api_client.MondayTemplateConfig{
		Mapping: json.RawMessage(`{ "status_14": { "value": "0" } }`),
	}
	var diags diag.Diagnostics
	encodeMappings(s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if s.MappingJSON.ValueString() != `{"status_14":{"value":"0"}}` {
		t.Errorf("expected compact normalized JSON, got %s", s.MappingJSON.ValueString())
	}
}

// An empty/absent API mapping must leave a null planned value null (no spurious "" diff).
func TestEncodeMappings_EmptyAPIKeepsNullPlan(t *testing.T) {
	s := &state{
		MappingJSON:             types.StringNull(),
		AlertStatusMappingJSON:  types.StringNull(),
		TicketStatusMappingJSON: types.StringNull(),
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

// A compact, key-sorted mapping must survive decode -> encode unchanged. Orca field
// references use the bare-string shorthand; literal objects (person) pass through.
func TestDecodeEncodeRoundTrip(t *testing.T) {
	orig := `{"long_text_mkn8v2sp":["alert_id"],"person":[{"value":{"id":"66396150","kind":"person"}}]}`
	s := &state{MappingJSON: types.StringValue(orig)}
	var cfg api_client.MondayTemplateConfig
	var diags diag.Diagnostics
	decodeMappings(s, &cfg, &diags)
	encodeMappings(s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if s.MappingJSON.ValueString() != orig {
		t.Errorf("round-trip drifted: got %s want %s", s.MappingJSON.ValueString(), orig)
	}
}
