package shift_left_integration

import (
	"encoding/json"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Assert Adopt() output marshals to the expected wire JSON.

func marshalToMap(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

func TestAdopt_ClearsProjectWhenConfigEmpty_WireFormat(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{FromConfig: types.StringValue("")},
		ExistingUnit{PolicyIDs: []string{"pol-1"}, ProjectID: "proj-old"},
	)
	got := marshalToMap(t, ad)
	// Cleared project_id must omit the key (omitempty), not send "".
	if _, present := got["project_id"]; present {
		t.Errorf("expected project_id key omitted from the wire body when cleared, got %v", got["project_id"])
	}
	policies, ok := got["policies"].([]any)
	if !ok || len(policies) != 1 || policies[0] != "pol-1" {
		t.Errorf("expected policies restored on the wire after clearing project, got %v", got["policies"])
	}
}

func TestAdopt_BindsProjectFromConfig_WireFormat(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{FromConfig: types.StringValue("proj-new")},
		ExistingUnit{PolicyIDs: []string{"pol-1"}},
	)
	got := marshalToMap(t, ad)
	if got["project_id"] != "proj-new" {
		t.Errorf("expected project_id on the wire, got %v", got["project_id"])
	}
	// Project-bound: policies key must be present and null (no omitempty).
	if v, present := got["policies"]; !present || v != nil {
		t.Errorf("expected policies present and null when project-bound, got %v (present=%v)", v, present)
	}
}

func TestAdopt_DefaultPoliciesClearsPolicies_WireFormat(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(true),
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-1")}),
		nil,
		ProjectIntent{},
		ExistingUnit{DefaultPolicies: false, PolicyIDs: []string{"pol-1", "pol-2"}},
	)
	got := marshalToMap(t, ad)
	if got["default_policies"] != true {
		t.Errorf("expected default_policies true on the wire, got %v", got["default_policies"])
	}
	policies, ok := got["policies"].([]any)
	if !ok || len(policies) != 0 {
		t.Errorf("expected an empty (not null, not omitted) policies array on the wire, got %v", got["policies"])
	}
}

func TestAdopt_RemapsLegacyScanAllMode_WireFormat(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolNull(),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{},
		ExistingUnit{InstallationMode: "SCAN_ALL", PolicyIDs: []string{"pol-1"}},
	)
	got := marshalToMap(t, ad)
	// installation_mode has no omitempty; key must always be present.
	if _, present := got["installation_mode"]; !present {
		t.Fatal("expected installation_mode key always present on the wire")
	}
	if got["installation_mode"] != "SELECTED_REPOSITORIES" {
		t.Errorf("expected legacy SCAN_ALL remapped on the wire, got %v", got["installation_mode"])
	}
}

func TestAdopt_MergesConfigSettings_WireFormat(t *testing.T) {
	overlay := &ConfigSettingsModel{
		PrSummaryComment: types.StringValue("ONLY_ON_FAILED_ISSUES"),
	}
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetNull(types.StringType),
		overlay,
		ProjectIntent{},
		ExistingUnit{
			ConfigSettings: api_client.ShiftLeftConfigSettings{
				CommentsOnPullRequests: "ALWAYS",
				PrSummaryComment:       "ALWAYS",
				SkipCheckRuns:          "ALWAYS",
				ConfigFileSupport:      "ENABLED",
			},
		},
	)
	got := marshalToMap(t, ad)
	cs, ok := got["configuration_settings"].(map[string]any)
	if !ok {
		t.Fatalf("configuration_settings missing or wrong shape: %v", got["configuration_settings"])
	}
	if cs["pr_summary_comment"] != "ONLY_ON_FAILED_ISSUES" {
		t.Errorf("expected overlay pr_summary_comment on the wire, got %v", cs["pr_summary_comment"])
	}
	if cs["comments_on_pull_requests"] != "ALWAYS" {
		t.Errorf("expected live comments_on_pull_requests preserved on the wire, got %v", cs["comments_on_pull_requests"])
	}
}
