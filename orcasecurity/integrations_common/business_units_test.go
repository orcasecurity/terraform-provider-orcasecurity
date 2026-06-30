package integrations_common

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBusinessUnitsFromAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("empty api and null planned stay null", func(t *testing.T) {
		got, diags := BusinessUnitsFromAPI(ctx, nil, types.SetNull(types.StringType))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if !got.IsNull() {
			t.Errorf("expected null set, got %v", got)
		}
	})

	t.Run("populated api returns set with elements", func(t *testing.T) {
		got, diags := BusinessUnitsFromAPI(ctx, []string{"a", "b"}, types.SetNull(types.StringType))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if got.IsNull() {
			t.Fatal("expected non-null set")
		}
		var out []string
		got.ElementsAs(ctx, &out, false)
		if len(out) != 2 {
			t.Errorf("expected 2 elements, got %d", len(out))
		}
	})
}

func TestBusinessUnitsToAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("null planned returns nil", func(t *testing.T) {
		var diags diag.Diagnostics
		got := BusinessUnitsToAPI(ctx, types.SetNull(types.StringType), &diags)
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("unknown planned returns nil", func(t *testing.T) {
		var diags diag.Diagnostics
		got := BusinessUnitsToAPI(ctx, types.SetUnknown(types.StringType), &diags)
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("populated planned returns slice", func(t *testing.T) {
		set, _ := types.SetValueFrom(ctx, types.StringType, []string{"x", "y"})
		var diags diag.Diagnostics
		got := BusinessUnitsToAPI(ctx, set, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if len(got) != 2 {
			t.Errorf("expected 2 elements, got %d", len(got))
		}
	})
}
