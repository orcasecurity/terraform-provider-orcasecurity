package integrations_common

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Bare strings expand to {"orca":...}; literal objects and non-list values pass through.
func TestDecodeOrcaMappingField_ExpandsShorthand(t *testing.T) {
	in := types.StringValue(`{"category":[{"value":"software"}],"literal":{"custom":"5"},"summary":["alert_name","alert_id"]}`)
	raw, diags := DecodeOrcaMappingField(in, "mapping_json")
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	want := `{"category":[{"value":"software"}],"literal":{"custom":"5"},"summary":[{"orca":"alert_name"},{"orca":"alert_id"}]}`
	if string(raw) != want {
		t.Errorf("expand mismatch:\n got %s\nwant %s", raw, want)
	}
}

// {"orca":...} collapses back to bare strings; literals and multi-key objects are kept.
func TestEncodeOrcaMappingField_CollapsesShorthand(t *testing.T) {
	api := []byte(`{"category":[{"value":"software"}],"summary":[{"orca":"alert_name"},{"orca":"alert_id"}]}`)
	out, diags := EncodeOrcaMappingField(api, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	want := `{"category":[{"value":"software"}],"summary":["alert_name","alert_id"]}`
	if out.ValueString() != want {
		t.Errorf("collapse mismatch:\n got %s\nwant %s", out.ValueString(), want)
	}
}

// The friendly HCL form must survive a decode -> API -> encode round-trip unchanged.
func TestOrcaMapping_RoundTrip(t *testing.T) {
	orig := `{"category":[{"value":"software"}],"summary":["alert_name","alert_id"]}`
	planned := types.StringValue(orig)
	raw, d1 := DecodeOrcaMappingField(planned, "mapping_json")
	out, d2 := EncodeOrcaMappingField(raw, planned)
	if d1.HasError() || d2.HasError() {
		t.Fatalf("unexpected diags: %v %v", d1, d2)
	}
	if out.ValueString() != orig {
		t.Errorf("round-trip drifted:\n got %s\nwant %s", out.ValueString(), orig)
	}
}

// A person-style object value ({"value":{...}}) must not be collapsed.
func TestEncodeOrcaMappingField_KeepsObjectValueLiteral(t *testing.T) {
	api := []byte(`{"person":[{"value":{"id":"66396150","kind":"person"}}]}`)
	out, diags := EncodeOrcaMappingField(api, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	want := `{"person":[{"value":{"id":"66396150","kind":"person"}}]}`
	if out.ValueString() != want {
		t.Errorf("should keep literal object:\n got %s\nwant %s", out.ValueString(), want)
	}
}

// Null / empty mapping stays null (no spurious diff).
func TestOrcaMappingField_EmptyStaysNull(t *testing.T) {
	raw, d := DecodeOrcaMappingField(types.StringNull(), "mapping_json")
	if d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	out, d2 := EncodeOrcaMappingField(raw, types.StringNull())
	if d2.HasError() {
		t.Fatalf("unexpected diags: %v", d2)
	}
	if !out.IsNull() {
		t.Errorf("expected null, got %v", out)
	}
}
