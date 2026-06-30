package integrations_common

import (
	"encoding/json"
	"testing"

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
		got, diags := EncodeJSONField(nil, types.StringNull())
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !got.IsNull() {
			t.Errorf("expected null, got %v", got)
		}
	})

	t.Run("empty raw with planned value preserves planned", func(t *testing.T) {
		planned := types.StringValue(`{"old":1}`)
		got, diags := EncodeJSONField(nil, planned)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got.ValueString() != `{"old":1}` {
			t.Errorf("expected planned preserved, got %q", got.ValueString())
		}
	})

	t.Run("valid raw re-marshals to compact json", func(t *testing.T) {
		raw := json.RawMessage(`{ "a" : 1 }`)
		got, diags := EncodeJSONField(raw, types.StringNull())
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got.ValueString() != `{"a":1}` {
			t.Errorf("expected compact json, got %q", got.ValueString())
		}
	})

	t.Run("malformed raw surfaces diagnostic", func(t *testing.T) {
		raw := json.RawMessage(`{not-json`)
		_, diags := EncodeJSONField(raw, types.StringNull())
		if !diags.HasError() {
			t.Error("expected error diagnostic for malformed json from API")
		}
	})
}
