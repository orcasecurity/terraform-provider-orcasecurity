package webhook_variant_common

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// apiObject is the response we'd get from /api/external_service/config. The test forces a
// mismatch between the planned config block and the API echo to confirm that:
//   - extractTopLevel (Create/Update) leaves the planned config in place
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

func TestExtractTopLevel_LeavesConfigBlockAlone(t *testing.T) {
	s := &state{
		WebhookURL: types.StringValue("https://planned.example.com"),
		APIKey:     types.StringValue("planned-key"),
		BodyFields: types.ListNull(types.StringType),
	}
	apiObj := sampleAPIResponse()

	obj := extractTopLevel(apiObj, s)

	if obj.ID != "id-1" {
		t.Errorf("expected ID extracted, got %q", obj.ID)
	}
	if s.WebhookURL.ValueString() != "https://planned.example.com" {
		t.Errorf("expected planned WebhookURL preserved, got %q", s.WebhookURL.ValueString())
	}
	if s.APIKey.ValueString() != "planned-key" {
		t.Errorf("expected planned APIKey preserved, got %q", s.APIKey.ValueString())
	}
}

func TestExtractFull_RefreshesConfigFromAPI(t *testing.T) {
	s := &state{
		WebhookURL: types.StringValue("https://stale-planned.example.com"),
		APIKey:     types.StringValue("planned-key"),
		BodyFields: types.ListNull(types.StringType),
	}
	apiObj := sampleAPIResponse()

	obj := extractFull(apiObj, s)

	if obj.ID != "id-1" {
		t.Errorf("expected ID extracted, got %q", obj.ID)
	}
	if s.WebhookURL.ValueString() != "https://api-echoed.example.com" {
		t.Errorf("expected WebhookURL refreshed from API, got %q", s.WebhookURL.ValueString())
	}
	// api_key intentionally not overwritten when the planned value is known.
	if s.APIKey.ValueString() != "planned-key" {
		t.Errorf("expected planned APIKey preserved (sensitive), got %q", s.APIKey.ValueString())
	}
	if s.BodyFields.IsNull() {
		t.Errorf("expected BodyFields populated from API, got null")
	}
}

func TestExtractFull_APIKeyUnknownTakesAPIValue(t *testing.T) {
	s := &state{
		WebhookURL: types.StringValue("https://planned.example.com"),
		APIKey:     types.StringUnknown(),
		BodyFields: types.ListNull(types.StringType),
	}
	apiObj := sampleAPIResponse()

	extractFull(apiObj, s)

	if s.APIKey.ValueString() != "echoed-key" {
		t.Errorf("expected API key copied when planned is unknown, got %q", s.APIKey.ValueString())
	}
}

// Sanity check the Spec wiring — both Extract and ExtractOnRead must produce the same
// cross-variant APIObject so the base loop sees identical id / business_units regardless of
// which hook ran.
func TestExtractHooks_AgreeOnCommon(t *testing.T) {
	apiObj := sampleAPIResponse()

	topLevel := extractTopLevel(apiObj, &state{})
	full := extractFull(apiObj, &state{
		WebhookURL: types.StringNull(),
		APIKey:     types.StringNull(),
		BodyFields: types.ListNull(types.StringType),
	})

	if topLevel.ID != full.ID || topLevel.TemplateName != full.TemplateName {
		t.Errorf("Extract vs ExtractOnRead disagree on common fields: %+v vs %+v", topLevel, full)
	}
	if len(topLevel.BusinessUnits) != len(full.BusinessUnits) {
		t.Errorf("business_units mismatch between hooks: %v vs %v", topLevel.BusinessUnits, full.BusinessUnits)
	}
	_ = topLevel
}
