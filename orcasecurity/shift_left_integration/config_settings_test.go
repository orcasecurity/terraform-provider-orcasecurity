package shift_left_integration

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
		ArchiveConditions:       types.ListValueMust(types.StringType, []attr.Value{types.StringValue("AVOID_SCAN")}),
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
		UnavailableConditions: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("DELETE_REPO")}),
	}
	api := ExpandConfigSettings(m)
	if api.InstallationReposConfig == nil || api.InstallationReposConfig.ArchiveActions != nil {
		t.Fatalf("expected only UnavailableActions to be set: %+v", api.InstallationReposConfig)
	}
	if len(api.InstallationReposConfig.UnavailableActions.Conditions) != 1 || api.InstallationReposConfig.UnavailableActions.Conditions[0] != "DELETE_REPO" {
		t.Fatalf("expand dropped unavailable conditions: %+v", api.InstallationReposConfig)
	}

	back := FlattenConfigSettings(api)
	unavailable := back.UnavailableConditions.Elements()
	if len(unavailable) != 1 || unavailable[0].(types.String).ValueString() != "DELETE_REPO" {
		t.Fatalf("flatten dropped unavailable conditions: %+v", back.UnavailableConditions)
	}
	if !back.ArchiveConditions.IsNull() {
		t.Fatalf("expected null ArchiveConditions, got: %+v", back.ArchiveConditions)
	}
}

func TestConfigSettingsAttributes_ArchiveAlwaysPresent(t *testing.T) {
	attrs := ConfigSettingsAttributes()
	// base fields always present
	for _, key := range []string{"disable_scan_pull_requests", "comments_on_pull_requests", "pr_summary_comment", "skip_check_runs", "config_file_support", "pr_summary_appendix", "archive_conditions", "unavailable_conditions"} {
		if _, ok := attrs[key]; !ok {
			t.Fatalf("expected field %q to always be present", key)
		}
	}

	for _, k := range []string{"archive_conditions", "unavailable_conditions"} {
		l, ok := attrs[k].(schema.ListAttribute)
		if !ok || !l.Optional || !l.Computed {
			t.Fatalf("%s must be Optional+Computed, got: %+v", k, attrs[k])
		}
	}
}

func TestConfigSettingsAttributes_OptionalComputed(t *testing.T) {
	attrs := ConfigSettingsAttributes()
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
	if !back.ArchiveConditions.IsNull() || !back.UnavailableConditions.IsNull() {
		t.Fatalf("expected null condition lists, got: %+v / %+v", back.ArchiveConditions, back.UnavailableConditions)
	}
}
