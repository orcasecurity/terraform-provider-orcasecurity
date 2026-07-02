package webhook_variant_common

import (
	"context"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCustomHeadersToAPI_NullMap(t *testing.T) {
	ctx := context.Background()
	got, diags := CustomHeadersToAPI(ctx, types.MapNull(CustomHeaderListType()))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got != nil {
		t.Errorf("expected nil map for null planned, got %v", got)
	}
}

func TestCustomHeadersToAPI_RoundTrip(t *testing.T) {
	ctx := context.Background()
	objType := CustomHeaderObjectType()

	mkObj := func(custom string) attr.Value {
		obj, _ := types.ObjectValue(objType.AttrTypes, map[string]attr.Value{
			"custom": types.StringValue(custom),
		})
		return obj
	}

	listForAuth, _ := types.ListValue(objType, []attr.Value{mkObj("Bearer abc"), mkObj("Bearer def")})
	listForTrace, _ := types.ListValue(objType, []attr.Value{mkObj("trace-1")})
	planned, _ := types.MapValue(CustomHeaderListType(), map[string]attr.Value{
		"authorization": listForAuth,
		"x-trace":       listForTrace,
	})

	apiMap, diags := CustomHeadersToAPI(ctx, planned)
	if diags.HasError() {
		t.Fatalf("ToAPI diagnostics: %v", diags)
	}
	if len(apiMap["authorization"]) != 2 || apiMap["authorization"][0].Custom != "Bearer abc" {
		t.Errorf("authorization map malformed: %+v", apiMap["authorization"])
	}
	if len(apiMap["x-trace"]) != 1 || apiMap["x-trace"][0].Custom != "trace-1" {
		t.Errorf("x-trace map malformed: %+v", apiMap["x-trace"])
	}

	back, diags := CustomHeadersFromAPI(apiMap, planned)
	if diags.HasError() {
		t.Fatalf("FromAPI diagnostics: %v", diags)
	}
	if back.IsNull() {
		t.Fatal("expected non-null map after round-trip")
	}
	elements := back.Elements()
	if len(elements) != 2 {
		t.Errorf("expected 2 keys after round-trip, got %d", len(elements))
	}
}

func TestCustomHeadersFromAPI_NullPreservedOnEmpty(t *testing.T) {
	got, diags := CustomHeadersFromAPI(map[string][]api_client.WebhookCustomHeaderValue{}, types.MapNull(CustomHeaderListType()))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !got.IsNull() {
		t.Errorf("expected null preserved when API and planned both empty")
	}
}
