package slack

import (
	"context"
	"encoding/json"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mapVal builds a types.Map matching the `mapping` attribute shape from a Go map.
func mapVal(t *testing.T, m map[string][]string) types.Map {
	t.Helper()
	elems := make(map[string]attr.Value, len(m))
	for section, fields := range m {
		vals := make([]attr.Value, 0, len(fields))
		for _, f := range fields {
			vals = append(vals, types.StringValue(f))
		}
		lv, d := types.ListValue(types.StringType, vals)
		if d.HasError() {
			t.Fatalf("list build: %v", d)
		}
		elems[section] = lv
	}
	mv, d := types.MapValue(mappingElemType, elems)
	if d.HasError() {
		t.Fatalf("map build: %v", d)
	}
	return mv
}

// decodeMapping must wrap each field name into a {"orca": <field>} entry per section.
func TestDecodeMapping_WrapsOrcaEntries(t *testing.T) {
	s := &state{Mapping: mapVal(t, map[string][]string{
		"title":       {"alert_id", "orca_score"},
		"description": {"details"},
	})}
	var cfg api_client.SlackConfig
	var diags diag.Diagnostics
	decodeMapping(context.Background(), s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	var got map[string][]map[string]string
	if err := json.Unmarshal(cfg.Mapping, &got); err != nil {
		t.Fatalf("bad wire json: %v", err)
	}
	want := map[string][]map[string]string{
		"title":       {{"orca": "alert_id"}, {"orca": "orca_score"}},
		"description": {{"orca": "details"}},
	}
	if len(got) != len(want) || len(got["title"]) != 2 || got["title"][0]["orca"] != "alert_id" || got["description"][0]["orca"] != "details" {
		t.Errorf("wire mapping mismatch: %s", cfg.Mapping)
	}
}

// encodeMapping must unwrap {"orca": <field>} entries back into plain field-name lists.
func TestEncodeMapping_UnwrapsOrcaEntries(t *testing.T) {
	s := &state{}
	cfg := api_client.SlackConfig{
		Mapping: json.RawMessage(`{"title":[{"orca":"alert_id"},{"orca":"orca_score"}],"description":[{"orca":"details"}]}`),
	}
	var diags diag.Diagnostics
	encodeMapping(context.Background(), s, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	var got map[string][]string
	if d := s.Mapping.ElementsAs(context.Background(), &got, false); d.HasError() {
		t.Fatalf("elements as: %v", d)
	}
	if len(got["title"]) != 2 || got["title"][0] != "alert_id" || got["description"][0] != "details" {
		t.Errorf("unwrapped mapping mismatch: %v", got)
	}
}

// A non-orca mapping entry from the API must surface an error, not be silently dropped.
func TestEncodeMapping_NonOrcaEntryErrors(t *testing.T) {
	s := &state{}
	cfg := api_client.SlackConfig{
		Mapping: json.RawMessage(`{"title":[{"custom":"literal"}]}`),
	}
	var diags diag.Diagnostics
	encodeMapping(context.Background(), s, &cfg, &diags)
	if !diags.HasError() {
		t.Fatal("expected error diag for non-orca mapping entry")
	}
}

// A decode -> encode round-trip must preserve section field lists.
func TestDecodeEncodeRoundTrip(t *testing.T) {
	s := &state{Mapping: mapVal(t, map[string][]string{
		"title":       {"alert_id", "orca_score", "source"},
		"description": {"details", "findings"},
	})}
	var cfg api_client.SlackConfig
	var diags diag.Diagnostics
	decodeMapping(context.Background(), s, &cfg, &diags)
	out := &state{}
	encodeMapping(context.Background(), out, &cfg, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	var got map[string][]string
	if d := out.Mapping.ElementsAs(context.Background(), &got, false); d.HasError() {
		t.Fatalf("elements as: %v", d)
	}
	if len(got["title"]) != 3 || got["title"][2] != "source" || len(got["description"]) != 2 {
		t.Errorf("round-trip drifted: %v", got)
	}
}
