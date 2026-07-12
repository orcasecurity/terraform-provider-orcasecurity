package integrations_common

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDecodeJSONField(t *testing.T) {
	t.Run("null planned returns nil", func(t *testing.T) {
		got, diags := DecodeJSONField(types.StringNull(), "field")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got != nil {
			t.Errorf("expected nil, got %s", got)
		}
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		got, diags := DecodeJSONField(types.StringValue(""), "field")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got != nil {
			t.Errorf("expected nil, got %s", got)
		}
	})

	t.Run("valid json passes through", func(t *testing.T) {
		got, diags := DecodeJSONField(types.StringValue(`{"a":1}`), "field")
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if string(got) != `{"a":1}` {
			t.Errorf("expected %q, got %q", `{"a":1}`, string(got))
		}
	})

	t.Run("invalid json surfaces diagnostic", func(t *testing.T) {
		_, diags := DecodeJSONField(types.StringValue("not json"), "field")
		if !diags.HasError() {
			t.Error("expected error diagnostic for invalid json")
		}
	})
}

func TestEncodeJSONField(t *testing.T) {
	t.Run("empty raw with null planned stays null", func(t *testing.T) {
		got, diags := EncodeJSONField(nil, jsontypes.NewNormalizedNull())
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !got.IsNull() {
			t.Errorf("expected null, got %v", got)
		}
	})

	t.Run("empty raw with planned value preserves planned", func(t *testing.T) {
		planned := jsontypes.NewNormalizedValue(`{"old":1}`)
		got, diags := EncodeJSONField(nil, planned)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got.ValueString() != `{"old":1}` {
			t.Errorf("expected planned preserved, got %q", got.ValueString())
		}
	})

	t.Run("null planned (import) canonicalizes to compact key-sorted json", func(t *testing.T) {
		// Matches HCL jsonencode / Go json.Marshal output so imported state does not diff
		// against a jsonencode(...) config (plan-time comparison skips semantic equality).
		raw := json.RawMessage(`{ "open" : "10000", "closed" : "10002" }`)
		got, diags := EncodeJSONField(raw, jsontypes.NewNormalizedNull())
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got.ValueString() != `{"closed":"10002","open":"10000"}` {
			t.Errorf("expected compact key-sorted json, got %q", got.ValueString())
		}
	})

	t.Run("planned value: raw stored verbatim (semantic equality handles formatting)", func(t *testing.T) {
		raw := json.RawMessage(`{ "a" : 1 }`)
		got, diags := EncodeJSONField(raw, jsontypes.NewNormalizedValue(`{"a":1}`))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got.ValueString() != `{ "a" : 1 }` {
			t.Errorf("expected raw stored verbatim, got %q", got.ValueString())
		}
	})

	t.Run("malformed raw surfaces diagnostic", func(t *testing.T) {
		raw := json.RawMessage(`{not-json`)
		_, diags := EncodeJSONField(raw, jsontypes.NewNormalizedNull())
		if !diags.HasError() {
			t.Error("expected error diagnostic for malformed json from API")
		}
	})
}
