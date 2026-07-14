package integrations_common

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringSliceFromSet(t *testing.T) {
	ctx := context.Background()

	t.Run("null set yields non-nil empty slice", func(t *testing.T) {
		out, diags := StringSliceFromSet(ctx, types.SetNull(types.StringType))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if out == nil || len(out) != 0 {
			t.Errorf("expected non-nil empty slice, got %#v", out)
		}
	})

	t.Run("unknown set yields non-nil empty slice", func(t *testing.T) {
		out, diags := StringSliceFromSet(ctx, types.SetUnknown(types.StringType))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if out == nil || len(out) != 0 {
			t.Errorf("expected non-nil empty slice, got %#v", out)
		}
	})

	t.Run("values pass through", func(t *testing.T) {
		set, _ := types.SetValueFrom(ctx, types.StringType, []string{"a", "b"})
		out, diags := StringSliceFromSet(ctx, set)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if len(out) != 2 {
			t.Errorf("expected 2 elements, got %#v", out)
		}
	})
}

func TestOptionalSetMatchPlan(t *testing.T) {
	ctx := context.Background()

	t.Run("empty API with null prior stays null", func(t *testing.T) {
		out, diags := OptionalSetMatchPlan(ctx, types.SetNull(types.StringType), nil)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !out.IsNull() {
			t.Errorf("expected null set, got %v", out)
		}
	})

	t.Run("empty API with configured prior yields empty set", func(t *testing.T) {
		prior, _ := types.SetValueFrom(ctx, types.StringType, []string{})
		out, diags := OptionalSetMatchPlan(ctx, prior, nil)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if out.IsNull() || len(out.Elements()) != 0 {
			t.Errorf("expected empty set, got %v", out)
		}
	})

	t.Run("API values win", func(t *testing.T) {
		out, diags := OptionalSetMatchPlan(ctx, types.SetNull(types.StringType), []string{"x"})
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if len(out.Elements()) != 1 {
			t.Errorf("expected 1 element, got %v", out)
		}
	})
}
