package shift_left_integration

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestConfigSettingsRoundTrip(t *testing.T) {
	m := &ConfigSettingsModel{
		DisableScanPullRequests: types.BoolValue(false),
		CommentsOnPullRequests:  types.StringValue("ONLY_ON_FAILED_ISSUES"),
		PrSummaryComment:        types.StringValue("ONLY_ON_FAILED_SCAN"),
		ConfigFileSupport:       types.StringValue("ENABLED"),
		PrSummaryAppendix:       types.StringValue("note"),
		ArchiveConditions:       []types.String{types.StringValue("AVOID_SCAN")},
	}
	api := ExpandConfigSettings(m)
	if api.CommentsOnPullRequests != "ONLY_ON_FAILED_ISSUES" || api.PrSummaryComment != "ONLY_ON_FAILED_SCAN" {
		t.Fatalf("expand lost enum values: %+v", api)
	}
	if api.InstallationReposConfig == nil || len(api.InstallationReposConfig.ArchiveActions.Conditions) != 1 {
		t.Fatalf("expand dropped archive conditions: %+v", api)
	}
	back := FlattenConfigSettings(api)
	if !back.PrSummaryComment.Equal(m.PrSummaryComment) {
		t.Fatalf("flatten mismatch: %v vs %v", back.PrSummaryComment, m.PrSummaryComment)
	}
}

func TestExpandConfigSettings_NoConditionsOmitsInstallationReposConfig(t *testing.T) {
	m := &ConfigSettingsModel{
		DisableScanPullRequests: types.BoolValue(true),
		ConfigFileSupport:       types.StringValue("DISABLED"),
	}
	api := ExpandConfigSettings(m)
	if api.InstallationReposConfig != nil {
		t.Fatalf("expected nil InstallationReposConfig when no conditions set, got: %+v", api.InstallationReposConfig)
	}
}

func TestExpandConfigSettings_UnavailableConditionsOnly(t *testing.T) {
	m := &ConfigSettingsModel{
		UnavailableConditions: []types.String{types.StringValue("DELETE_REPO")},
	}
	api := ExpandConfigSettings(m)
	if api.InstallationReposConfig == nil || api.InstallationReposConfig.ArchiveActions != nil {
		t.Fatalf("expected only UnavailableActions to be set: %+v", api.InstallationReposConfig)
	}
	if len(api.InstallationReposConfig.UnavailableActions.Conditions) != 1 || api.InstallationReposConfig.UnavailableActions.Conditions[0] != "DELETE_REPO" {
		t.Fatalf("expand dropped unavailable conditions: %+v", api.InstallationReposConfig)
	}

	back := FlattenConfigSettings(api)
	if len(back.UnavailableConditions) != 1 || back.UnavailableConditions[0].ValueString() != "DELETE_REPO" {
		t.Fatalf("flatten dropped unavailable conditions: %+v", back.UnavailableConditions)
	}
	if back.ArchiveConditions != nil {
		t.Fatalf("expected nil ArchiveConditions, got: %+v", back.ArchiveConditions)
	}
}

func TestConfigSettingsAttributes_FieldGating(t *testing.T) {
	allOff := ConfigSettingsAttributes(FieldGate{})
	if _, ok := allOff["archive_conditions"]; ok {
		t.Fatalf("expected archive_conditions to be omitted when ArchiveActions gate is off")
	}
	if _, ok := allOff["unavailable_conditions"]; ok {
		t.Fatalf("expected unavailable_conditions to be omitted when ArchiveActions gate is off")
	}
	// base fields always present
	for _, key := range []string{"disable_scan_pull_requests", "comments_on_pull_requests", "pr_summary_comment", "skip_check_runs", "config_file_support", "pr_summary_appendix"} {
		if _, ok := allOff[key]; !ok {
			t.Fatalf("expected base field %q to always be present", key)
		}
	}

	allOn := ConfigSettingsAttributes(FieldGate{ArchiveActions: true})
	for _, key := range []string{"skip_check_runs", "archive_conditions", "unavailable_conditions"} {
		if _, ok := allOn[key]; !ok {
			t.Fatalf("expected gated field %q to be present when gate is on", key)
		}
	}
}

func TestConfigSettingsAttributes_SkipCheckRunsAlwaysPresent(t *testing.T) {
	attrs := ConfigSettingsAttributes(FieldGate{ArchiveActions: false})
	if _, ok := attrs["skip_check_runs"]; !ok {
		t.Fatal("skip_check_runs must be present regardless of gate")
	}
	if _, ok := attrs["archive_conditions"]; ok {
		t.Fatal("archive_conditions must be absent when ArchiveActions=false")
	}
}

func TestConfigSettingsAttributes_ArchiveGate(t *testing.T) {
	attrs := ConfigSettingsAttributes(FieldGate{ArchiveActions: true})
	for _, k := range []string{"archive_conditions", "unavailable_conditions"} {
		if _, ok := attrs[k]; !ok {
			t.Fatalf("%s must be present when ArchiveActions=true", k)
		}
	}
}

func TestConfigSettingsAttributes_OptionalComputed(t *testing.T) {
	attrs := ConfigSettingsAttributes(FieldGate{ArchiveActions: true})
	b, ok := attrs["disable_scan_pull_requests"].(schema.BoolAttribute)
	if !ok || !b.Optional || !b.Computed {
		t.Fatal("disable_scan_pull_requests must be Optional+Computed")
	}
}

func TestFlattenConfigSettings_EmptyStringsBecomeNull(t *testing.T) {
	back := FlattenConfigSettings(api_client.ShiftLeftConfigSettings{})
	if !back.CommentsOnPullRequests.IsNull() {
		t.Fatalf("expected null CommentsOnPullRequests, got: %v", back.CommentsOnPullRequests)
	}
	if !back.PrSummaryComment.IsNull() {
		t.Fatalf("expected null PrSummaryComment, got: %v", back.PrSummaryComment)
	}
	if back.ArchiveConditions != nil || back.UnavailableConditions != nil {
		t.Fatalf("expected nil condition slices, got: %+v / %+v", back.ArchiveConditions, back.UnavailableConditions)
	}
}
