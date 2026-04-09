package group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOptionalUsersSetMatchPlan(t *testing.T) {
	ctx := context.Background()
	t.Run("api_nonempty_always_from_api", func(t *testing.T) {
		got, diags := optionalUsersSetMatchPlan(ctx, types.SetNull(types.StringType), []string{"a", "b"})
		if diags.HasError() {
			t.Fatal(diags)
		}
		if got.IsNull() || got.IsUnknown() {
			t.Fatalf("expected known set, got %#v", got)
		}
		if n := len(got.Elements()); n != 2 {
			t.Fatalf("elements: %d", n)
		}
	})
	t.Run("api_empty_and_ref_null_stays_null", func(t *testing.T) {
		got, diags := optionalUsersSetMatchPlan(ctx, types.SetNull(types.StringType), nil)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if !got.IsNull() {
			t.Fatalf("expected null, got %#v", got)
		}
	})
	t.Run("api_empty_and_ref_known_empty_stays_empty_set", func(t *testing.T) {
		empty, d := types.SetValueFrom(ctx, types.StringType, []string{})
		if d.HasError() {
			t.Fatal(d)
		}
		got, diags := optionalUsersSetMatchPlan(ctx, empty, []string{})
		if diags.HasError() {
			t.Fatal(diags)
		}
		if got.IsNull() || got.IsUnknown() {
			t.Fatalf("expected empty set, got %#v", got)
		}
		if n := len(got.Elements()); n != 0 {
			t.Fatalf("expected 0 elements, got %d", n)
		}
	})
	t.Run("api_empty_and_ref_had_members_becomes_empty_set", func(t *testing.T) {
		withUser, d := types.SetValueFrom(ctx, types.StringType, []string{"u1"})
		if d.HasError() {
			t.Fatal(d)
		}
		got, diags := optionalUsersSetMatchPlan(ctx, withUser, nil)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if got.IsNull() {
			t.Fatal("expected empty set after API cleared users, not null")
		}
		if n := len(got.Elements()); n != 0 {
			t.Fatalf("expected 0 elements, got %d", n)
		}
	})
}
