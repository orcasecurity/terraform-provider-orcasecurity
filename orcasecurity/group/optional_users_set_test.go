package group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func fatalIfDiags(t *testing.T, d diag.Diagnostics) {
	t.Helper()
	if d.HasError() {
		t.Fatal(d)
	}
}

func mustStringSet(t *testing.T, ctx context.Context, elems []string) types.Set {
	t.Helper()
	s, d := types.SetValueFrom(ctx, types.StringType, elems)
	fatalIfDiags(t, d)
	return s
}

func assertKnownSetElementCount(t *testing.T, got types.Set, want int) {
	t.Helper()
	if got.IsNull() || got.IsUnknown() {
		t.Fatalf("expected known set, got %#v", got)
	}
	if n := len(got.Elements()); n != want {
		t.Fatalf("elements: want %d got %d", want, n)
	}
}

func assertNullSet(t *testing.T, got types.Set) {
	t.Helper()
	if !got.IsNull() {
		t.Fatalf("expected null, got %#v", got)
	}
}

func assertEmptySetAfterAPIClear(t *testing.T, got types.Set) {
	t.Helper()
	if got.IsNull() {
		t.Fatal("expected empty set after API cleared users, not null")
	}
	if n := len(got.Elements()); n != 0 {
		t.Fatalf("expected 0 elements, got %d", n)
	}
}

func testOptionalUsersSet_nonemptyAlwaysFromAPI(t *testing.T) {
	ctx := context.Background()
	got, diags := optionalUsersSetMatchPlan(ctx, types.SetNull(types.StringType), []string{"a", "b"})
	fatalIfDiags(t, diags)
	assertKnownSetElementCount(t, got, 2)
}

func testOptionalUsersSet_emptyRefNullStaysNull(t *testing.T) {
	ctx := context.Background()
	got, diags := optionalUsersSetMatchPlan(ctx, types.SetNull(types.StringType), nil)
	fatalIfDiags(t, diags)
	assertNullSet(t, got)
}

func testOptionalUsersSet_emptyRefKnownEmptyStaysEmptySet(t *testing.T) {
	ctx := context.Background()
	empty := mustStringSet(t, ctx, []string{})
	got, diags := optionalUsersSetMatchPlan(ctx, empty, []string{})
	fatalIfDiags(t, diags)
	assertKnownSetElementCount(t, got, 0)
}

func testOptionalUsersSet_emptyAfterHadMembers(t *testing.T) {
	ctx := context.Background()
	withUser := mustStringSet(t, ctx, []string{"u1"})
	got, diags := optionalUsersSetMatchPlan(ctx, withUser, nil)
	fatalIfDiags(t, diags)
	assertEmptySetAfterAPIClear(t, got)
}

func TestOptionalUsersSetMatchPlan(t *testing.T) {
	t.Run("api_nonempty_always_from_api", testOptionalUsersSet_nonemptyAlwaysFromAPI)
	t.Run("api_empty_and_ref_null_stays_null", testOptionalUsersSet_emptyRefNullStaysNull)
	t.Run("api_empty_and_ref_known_empty_stays_empty_set", testOptionalUsersSet_emptyRefKnownEmptyStaysEmptySet)
	t.Run("api_empty_and_ref_had_members_becomes_empty_set", testOptionalUsersSet_emptyAfterHadMembers)
}
