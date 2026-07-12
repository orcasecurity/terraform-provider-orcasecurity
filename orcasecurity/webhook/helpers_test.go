package webhook

import (
	"context"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// sampleAPIResponse is the response we'd get from /api/external_service/config. The extract
// tests force a mismatch between the planned config block and the API echo to confirm that:
//   - ExtractTopLevel (Create/Update) leaves the planned config in place
//   - extractFull (Read) refreshes the config block from the API
func sampleAPIResponse() *api_client.WebhookExternalServiceConfig {
	return &api_client.WebhookExternalServiceConfig{
		ID:           "id-1",
		TemplateName: "tpl",
		IsEnabled:    true,
		IsDefault:    false,
		Config: api_client.WebhookResourceConfig{
			WebhookURL: "https://api-echoed.example.com",
			Type:       "coralogix",
			APIKey:     "echoed-key",
			BodyFields: []string{"a", "b"},
		},
		BusinessUnits: []string{"bu-1"},
	}
}

func plannedState(url, key string) *state {
	return &state{
		Config: &webhookConfigModel{
			WebhookURL: types.StringValue(url),
			APIKey:     types.StringValue(key),
			BodyFields: types.ListNull(types.StringType),
		},
	}
}

func TestExtractTopLevel_LeavesConfigBlockAlone(t *testing.T) {
	s := plannedState("https://planned.example.com", "planned-key")
	apiObj := sampleAPIResponse()

	obj := ExtractTopLevel(apiObj, s, &diag.Diagnostics{})

	if obj.ID != "id-1" {
		t.Errorf("expected ID extracted, got %q", obj.ID)
	}
	if s.Config.WebhookURL.ValueString() != "https://planned.example.com" {
		t.Errorf("expected planned WebhookURL preserved, got %q", s.Config.WebhookURL.ValueString())
	}
	if s.Config.APIKey.ValueString() != "planned-key" {
		t.Errorf("expected planned APIKey preserved, got %q", s.Config.APIKey.ValueString())
	}
}

func TestExtractFull_RefreshesConfigFromAPI(t *testing.T) {
	s := plannedState("https://stale-planned.example.com", "planned-key")
	apiObj := sampleAPIResponse()

	obj := extractFull(apiObj, s, &diag.Diagnostics{})

	if obj.ID != "id-1" {
		t.Errorf("expected ID extracted, got %q", obj.ID)
	}
	if s.Config.WebhookURL.ValueString() != "https://api-echoed.example.com" {
		t.Errorf("expected WebhookURL refreshed from API, got %q", s.Config.WebhookURL.ValueString())
	}
	// api_key intentionally not overwritten when the planned value is known.
	if s.Config.APIKey.ValueString() != "planned-key" {
		t.Errorf("expected planned APIKey preserved (sensitive), got %q", s.Config.APIKey.ValueString())
	}
	if s.Config.BodyFields.IsNull() {
		t.Errorf("expected BodyFields populated from API, got null")
	}
}

func TestExtractFull_APIKeyUnknownTakesAPIValue(t *testing.T) {
	s := &state{
		Config: &webhookConfigModel{
			WebhookURL: types.StringValue("https://planned.example.com"),
			APIKey:     types.StringUnknown(),
			BodyFields: types.ListNull(types.StringType),
		},
	}
	apiObj := sampleAPIResponse()

	extractFull(apiObj, s, &diag.Diagnostics{})

	if s.Config.APIKey.ValueString() != "echoed-key" {
		t.Errorf("expected API key copied when planned is unknown, got %q", s.Config.APIKey.ValueString())
	}
}

// Sanity check the Spec wiring — both Extract and ExtractOnRead must produce the same
// cross-cutting APIObject so the base loop sees identical id / business_units regardless of
// which hook ran.
func TestExtractHooks_AgreeOnCommon(t *testing.T) {
	apiObj := sampleAPIResponse()

	topLevel := ExtractTopLevel(apiObj, &state{}, &diag.Diagnostics{})
	full := extractFull(apiObj, plannedState("https://planned.example.com", "planned-key"), &diag.Diagnostics{})

	if topLevel.ID != full.ID || topLevel.TemplateName != full.TemplateName {
		t.Errorf("Extract vs ExtractOnRead disagree on common fields: %+v vs %+v", topLevel, full)
	}
	if len(topLevel.BusinessUnits) != len(full.BusinessUnits) {
		t.Errorf("business_units mismatch between hooks: %v vs %v", topLevel.BusinessUnits, full.BusinessUnits)
	}
}

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
