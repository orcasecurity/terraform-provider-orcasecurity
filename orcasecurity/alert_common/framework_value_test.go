package alert_common

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFrameworksToListUsesEmptyListForNoLinks(t *testing.T) {
	list, diags := FrameworksToList(context.Background(), nil)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if list.IsNull() || list.IsUnknown() {
		t.Fatalf("no links must read back as a known empty list, got %v", list)
	}
	if got := len(list.Elements()); got != 0 {
		t.Fatalf("expected 0 elements, got %d", got)
	}
}

func TestFrameworksRoundTrip(t *testing.T) {
	ctx := context.Background()
	want := []Framework{{
		Name:     types.StringValue("Bayer GCP CSR"),
		Section:  types.StringValue("1. AM - Asset Management"),
		Priority: types.StringValue("medium"),
	}}

	list, diags := FrameworksToList(ctx, want)
	if diags.HasError() {
		t.Fatal(diags)
	}
	got, diags := FrameworksFromList(ctx, list)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("round trip changed the value: %v", got)
	}
	if FrameworksCount(list) != 1 {
		t.Fatalf("expected count 1, got %d", FrameworksCount(list))
	}
}

// A null or unknown plan value means the config says nothing about the links.
// Turning either into frameworks would send a clearing PUT (WASP-1672).
func TestFrameworksFromListIgnoresNullAndUnknown(t *testing.T) {
	ctx := context.Background()
	for name, list := range map[string]types.List{
		"null":    types.ListNull(FrameworkObjectType()),
		"unknown": types.ListUnknown(FrameworkObjectType()),
	} {
		t.Run(name, func(t *testing.T) {
			got, diags := FrameworksFromList(ctx, list)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if got != nil {
				t.Fatalf("expected no frameworks, got %v", got)
			}
			if FrameworksCount(list) != 0 {
				t.Fatalf("expected count 0, got %d", FrameworksCount(list))
			}
		})
	}
}
