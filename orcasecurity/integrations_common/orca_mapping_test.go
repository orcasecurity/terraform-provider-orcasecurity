package integrations_common

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// semanticEqual is a tiny helper: build two OrcaMapping values and ask whether the framework's
// semantic-equality hook considers them equal. state is the value already in state (v); cfg is
// the incoming config/plan value (newValuable).
func semanticEqual(t *testing.T, state, cfg string) bool {
	t.Helper()
	eq, diags := NewOrcaMappingValue(state).StringSemanticEquals(context.Background(), NewOrcaMappingValue(cfg))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	return eq
}

// The whole point of the shorthand: state holds the API's {"orca":"x"} form, config may use the
// bare-string shorthand. Semantic equality must treat them as equal so there is no perpetual diff.
func TestOrcaMapping_SemanticEquals_ShorthandMatchesObjectForm(t *testing.T) {
	apiForm := `{"summary":[{"orca":"alert_name"},{"orca":"alert_id"}]}`
	shorthand := `{"summary":["alert_name","alert_id"]}`
	if !semanticEqual(t, apiForm, shorthand) {
		t.Error("bare-string shorthand config should equal API {\"orca\":...} state")
	}
}

// The documented object form must also match the API state (no shorthand involved).
func TestOrcaMapping_SemanticEquals_ObjectFormMatches(t *testing.T) {
	apiForm := `{"summary":[{"orca":"alert_name"}]}`
	objectForm := `{"summary":[{"orca":"alert_name"}]}`
	if !semanticEqual(t, apiForm, objectForm) {
		t.Error("object-form config should equal API state")
	}
}

// Whitespace and object key order are inconsequential.
func TestOrcaMapping_SemanticEquals_IgnoresWhitespaceAndKeyOrder(t *testing.T) {
	state := `{"a":[{"orca":"x"}],"b":[{"value":"lit"}]}`
	cfg := `{
		"b": [ { "value": "lit" } ],
		"a": [ "x" ]
	}`
	if !semanticEqual(t, state, cfg) {
		t.Error("whitespace/key-order/shorthand differences should be equal")
	}
}

// A genuine difference must still register as not equal.
func TestOrcaMapping_SemanticEquals_DetectsRealDifference(t *testing.T) {
	if semanticEqual(t, `{"summary":["alert_name"]}`, `{"summary":["alert_id"]}`) {
		t.Error("different mappings must not be semantically equal")
	}
}

// Bare strings expand to {"orca":...} for the API payload; literal objects and non-list values
// pass through. (Payload side still needs the full wire form.)
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

// On read we store the API value verbatim (no collapse); semantic equality reconciles it with
// whatever shorthand the user wrote.
func TestEncodeOrcaMappingField_StoresApiValue(t *testing.T) {
	api := []byte(`{"summary":[{"orca":"alert_name"}]}`)
	out, diags := EncodeOrcaMappingField(api, NewOrcaMappingNull())
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if out.ValueString() != `{"summary":[{"orca":"alert_name"}]}` {
		t.Errorf("expected API value stored verbatim, got %q", out.ValueString())
	}
}

// Null / empty mapping stays null (no spurious diff against an unset attribute).
func TestOrcaMappingField_EmptyStaysNull(t *testing.T) {
	raw, d := DecodeOrcaMappingField(NewOrcaMappingNull(), "mapping_json")
	if d.HasError() {
		t.Fatalf("unexpected diags: %v", d)
	}
	out, d2 := EncodeOrcaMappingField(raw, NewOrcaMappingNull())
	if d2.HasError() {
		t.Fatalf("unexpected diags: %v", d2)
	}
	if !out.IsNull() {
		t.Errorf("expected null, got %v", out)
	}
}
