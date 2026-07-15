package jira_cloud_template

import (
	"context"
	"encoding/json"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// decodeMappings must copy the mapping shorthand into config.Mapping (expanded to wire form)
// and each status-map JSON string into its matching config RawMessage field.
func TestDecodeMappings_PopulatesConfigFromState(t *testing.T) {
	s := &state{
		MappingJSON:                    common.NewOrcaMappingValue(`{"summary":["alert_id"]}`),
		AlertStatusMappingJSON:         jsontypes.NewNormalizedValue(`{"in_progress":"10001"}`),
		TicketStatusMappingJSON:        jsontypes.NewNormalizedValue(`{"10000":{"status":"snoozed"}}`),
		SubtaskAlertStatusMappingJSON:  jsontypes.NewNormalizedValue(`{"in_progress":"20001"}`),
		SubtaskTicketStatusMappingJSON: jsontypes.NewNormalizedValue(`{"20000":{"status":"dismissed"}}`),
	}
	var cfg api_client.JiraCloudTemplateConfig
	var diags diag.Diagnostics
	decodeMappings(s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	// The bare-string shorthand is expanded into the {"orca":...} wire form the API expects.
	if string(cfg.Mapping) != `{"summary":[{"orca":"alert_id"}]}` {
		t.Errorf("mapping mismatch: %s", cfg.Mapping)
	}
	if string(cfg.AlertStatusMapping) != `{"in_progress":"10001"}` {
		t.Errorf("alert_status_mapping mismatch: %s", cfg.AlertStatusMapping)
	}
	if string(cfg.TicketStatusMapping) != `{"10000":{"status":"snoozed"}}` {
		t.Errorf("ticket_status_mapping mismatch: %s", cfg.TicketStatusMapping)
	}
	if string(cfg.SubtaskAlertStatusMapping) != `{"in_progress":"20001"}` {
		t.Errorf("subtask_alert_status_mapping mismatch: %s", cfg.SubtaskAlertStatusMapping)
	}
	if string(cfg.SubtaskTicketStatusMapping) != `{"20000":{"status":"dismissed"}}` {
		t.Errorf("subtask_ticket_status_mapping mismatch: %s", cfg.SubtaskTicketStatusMapping)
	}
}

// Null optional status maps must decode to nil RawMessages so json.Marshal omits them entirely
// (matching what the UI sends — unset, not empty).
func TestDecodeMappings_NullOptionalsStayNil(t *testing.T) {
	s := &state{
		MappingJSON:                    common.NewOrcaMappingValue(`{"summary":["alert_id"]}`),
		AlertStatusMappingJSON:         jsontypes.NewNormalizedNull(),
		TicketStatusMappingJSON:        jsontypes.NewNormalizedNull(),
		SubtaskAlertStatusMappingJSON:  jsontypes.NewNormalizedNull(),
		SubtaskTicketStatusMappingJSON: jsontypes.NewNormalizedNull(),
	}
	var cfg api_client.JiraCloudTemplateConfig
	var diags diag.Diagnostics
	decodeMappings(s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if cfg.AlertStatusMapping != nil {
		t.Errorf("expected nil alert_status_mapping, got %s", cfg.AlertStatusMapping)
	}
	if cfg.SubtaskTicketStatusMapping != nil {
		t.Errorf("expected nil subtask_ticket_status_mapping, got %s", cfg.SubtaskTicketStatusMapping)
	}
}

// Invalid JSON in the required mapping field must surface a plan-time diagnostic.
func TestDecodeMappings_InvalidMappingSurfacesError(t *testing.T) {
	s := &state{MappingJSON: common.NewOrcaMappingValue(`{not json`)}
	var cfg api_client.JiraCloudTemplateConfig
	var diags diag.Diagnostics
	decodeMappings(s, &cfg, &diags)
	if !diags.HasError() {
		t.Fatal("expected error diag for invalid mapping JSON")
	}
}

// Invalid JSON in an optional status-map field must surface a plan-time diagnostic.
func TestDecodeMappings_InvalidStatusMapSurfacesError(t *testing.T) {
	s := &state{
		MappingJSON:            common.NewOrcaMappingValue(`{"summary":["alert_id"]}`),
		AlertStatusMappingJSON: jsontypes.NewNormalizedValue(`{not json`),
	}
	var cfg api_client.JiraCloudTemplateConfig
	var diags diag.Diagnostics
	decodeMappings(s, &cfg, &diags)
	if !diags.HasError() {
		t.Fatal("expected error diag for invalid status-map JSON")
	}
}

// encodeMappings stores the API value verbatim; whitespace/key-order differences are absorbed
// by the mapping types' semantic equality, so the user's HCL form stays put (no diff).
func TestEncodeMappings_StoresApiValueSemanticallyEqual(t *testing.T) {
	planned := common.NewOrcaMappingValue(`{"summary":["alert_id"]}`)
	plannedAlert := jsontypes.NewNormalizedValue(`{"in_progress":"10001"}`)
	s := &state{MappingJSON: planned, AlertStatusMappingJSON: plannedAlert}
	cfg := api_client.JiraCloudTemplateConfig{
		Mapping:            json.RawMessage(`{ "summary": [ { "orca": "alert_id" } ] }`),
		AlertStatusMapping: json.RawMessage(`{ "in_progress": "10001" }`),
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
		t.Errorf("stored mapping should be semantically equal to plan, got %s", s.MappingJSON.ValueString())
	}
	eqA, dA := s.AlertStatusMappingJSON.StringSemanticEquals(context.Background(), plannedAlert)
	if dA.HasError() {
		t.Fatalf("unexpected diags: %v", dA)
	}
	if !eqA {
		t.Errorf("stored alert_status_mapping should be semantically equal to plan, got %s", s.AlertStatusMappingJSON.ValueString())
	}
}

// An empty/absent API mapping must leave a null planned value null (no spurious "" diff).
func TestEncodeMappings_EmptyAPIKeepsNullPlan(t *testing.T) {
	s := &state{
		MappingJSON:                    common.NewOrcaMappingNull(),
		AlertStatusMappingJSON:         jsontypes.NewNormalizedNull(),
		TicketStatusMappingJSON:        jsontypes.NewNormalizedNull(),
		SubtaskAlertStatusMappingJSON:  jsontypes.NewNormalizedNull(),
		SubtaskTicketStatusMappingJSON: jsontypes.NewNormalizedNull(),
	}
	var cfg api_client.JiraCloudTemplateConfig // all mapping fields nil
	var diags diag.Diagnostics
	encodeMappings(s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !s.MappingJSON.IsNull() {
		t.Errorf("expected null mapping, got %v", s.MappingJSON)
	}
	if !s.AlertStatusMappingJSON.IsNull() {
		t.Errorf("expected null alert_status_mapping, got %v", s.AlertStatusMappingJSON)
	}
}

// The friendly HCL form must round-trip decode -> API -> encode semantically unchanged, even
// though state ends up holding the API's expanded {"orca":...} wire form for the mapping.
func TestDecodeEncodeRoundTrip(t *testing.T) {
	origMapping := common.NewOrcaMappingValue(`{"summary":["alert_id"],"labels":[{"value":"orca"}]}`)
	origAlert := jsontypes.NewNormalizedValue(`{"in_progress":"10001"}`)
	s := &state{MappingJSON: origMapping, AlertStatusMappingJSON: origAlert}
	var cfg api_client.JiraCloudTemplateConfig
	var diags diag.Diagnostics
	decodeMappings(s, &cfg, &diags)
	// The encoder marshals with sorted keys, so labels precedes summary on the wire.
	if string(cfg.Mapping) != `{"labels":[{"value":"orca"}],"summary":[{"orca":"alert_id"}]}` {
		t.Errorf("payload not expanded to wire form: %s", cfg.Mapping)
	}
	encodeMappings(s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	eq, d := s.MappingJSON.StringSemanticEquals(context.Background(), origMapping)
	if d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	if !eq {
		t.Errorf("mapping round-trip drifted: got %s", s.MappingJSON.ValueString())
	}
	eqA, _ := s.AlertStatusMappingJSON.StringSemanticEquals(context.Background(), origAlert)
	if !eqA {
		t.Errorf("alert_status_mapping round-trip drifted: got %s", s.AlertStatusMappingJSON.ValueString())
	}
}

// The exported constructor must build without panicking and produce a non-nil resource wired to
// the generic config-integration spec (guards against a broken Spec definition).
func TestNewJiraCloudTemplateResource_Constructs(t *testing.T) {
	r := NewJiraCloudTemplateResource()
	if r == nil {
		t.Fatal("expected a non-nil resource")
	}
}

// variantAttributes must declare exactly the per-variant fields the state struct references, so
// a field added to state without a schema entry (or vice-versa) is caught here.
func TestVariantAttributes_DeclaresExpectedFields(t *testing.T) {
	attrs := variantAttributes()
	expected := []string{
		"resource_id", "resource_url", "project_id", "issue_type_id",
		"subtask_issue_type_id", "mapping_json", "alert_status_mapping_json",
		"ticket_status_mapping_json", "subtask_alert_status_mapping_json",
		"subtask_ticket_status_mapping_json", "business_units",
	}
	for _, name := range expected {
		if _, ok := attrs[name]; !ok {
			t.Errorf("variantAttributes missing %q", name)
		}
	}
	if len(attrs) != len(expected) {
		t.Errorf("variantAttributes has %d entries, expected %d", len(attrs), len(expected))
	}
}
