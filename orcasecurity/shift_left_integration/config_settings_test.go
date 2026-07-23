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
		PrSummaryComment:        types.StringValue("ONLY_ON_FAILED_ISSUES"),
		ConfigFileSupport:       types.StringValue("ENABLED"),
		PrSummaryAppendix:       types.StringValue("note"),
		ArchiveConditions:       types.ListValueMust(types.StringType, []attr.Value{types.StringValue("AVOID_SCAN")}),
	}
	api := ExpandConfigSettings(m)
	if api.CommentsOnPullRequests != "ONLY_ON_FAILED_ISSUES" || api.PrSummaryComment != "ONLY_ON_FAILED_ISSUES" {
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

func TestExpandConfigSettings_ExplicitEmptyListsClearsReposConfig(t *testing.T) {
	m := &ConfigSettingsModel{
		ArchiveConditions:     types.ListValueMust(types.StringType, []attr.Value{}),
		UnavailableConditions: types.ListValueMust(types.StringType, []attr.Value{}),
	}
	api := ExpandConfigSettings(m)
	if api.InstallationReposConfig == nil {
		t.Fatal("expected empty InstallationReposConfig object to clear server-side, got nil (omitted)")
	}
	if api.InstallationReposConfig.ArchiveActions != nil || api.InstallationReposConfig.UnavailableActions != nil {
		t.Fatalf("expected empty actions on clear, got: %+v", api.InstallationReposConfig)
	}
}

func TestMergeThenExpand_ClearArchiveConditions(t *testing.T) {
	base := FlattenConfigSettings(api_client.ShiftLeftConfigSettings{
		CommentsOnPullRequests: "ALWAYS",
		InstallationReposConfig: &api_client.ShiftLeftInstallationReposConfig{
			ArchiveActions:     &api_client.ShiftLeftArchiveActions{Conditions: []string{"AVOID_SCAN", "DELETE_REPO"}},
			UnavailableActions: &api_client.ShiftLeftArchiveActions{Conditions: []string{"DELETE_REPO"}},
		},
	})
	overlay := &ConfigSettingsModel{
		ArchiveConditions:     types.ListValueMust(types.StringType, []attr.Value{}),
		UnavailableConditions: types.ListValueMust(types.StringType, []attr.Value{}),
	}
	merged := MergeConfigSettings(base, overlay)
	api := ExpandConfigSettings(&merged)
	if api.InstallationReposConfig == nil {
		t.Fatal("clearing conditions must send explicit empty installation_repositories_configuration")
	}
	if api.CommentsOnPullRequests != "ALWAYS" {
		t.Fatalf("expected unrelated fields preserved, got comments=%q", api.CommentsOnPullRequests)
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

// TestMergeConfigSettings_NilOverlayReturnsBase covers adopt-existing Create
// calls where the user set no configuration_settings block at all.
func TestMergeConfigSettings_NilOverlayReturnsBase(t *testing.T) {
	base := ConfigSettingsModel{
		DisableScanPullRequests: types.BoolValue(true),
		CommentsOnPullRequests:  types.StringValue("ALWAYS"),
		PrSummaryComment:        types.StringValue("NEVER"),
		SkipCheckRuns:           types.StringValue("ALWAYS"),
		ConfigFileSupport:       types.StringValue("ENABLED"),
	}
	merged := MergeConfigSettings(base, nil)
	if !merged.CommentsOnPullRequests.Equal(base.CommentsOnPullRequests) {
		t.Fatalf("expected base returned unchanged, got: %+v", merged)
	}
}

// TestMergeConfigSettings_PartialOverlayWinsOnSetFieldsOnly is the core
// regression test for the "API requires a complete configuration_settings
// object" bug: a partial overlay (only pr_summary_comment explicitly set)
// must produce a fully-populated result where the overlay wins on the field
// it set and the base (current server) values are kept for every other
// field.
func TestMergeConfigSettings_PartialOverlayWinsOnSetFieldsOnly(t *testing.T) {
	base := ConfigSettingsModel{
		DisableScanPullRequests: types.BoolValue(false),
		CommentsOnPullRequests:  types.StringValue("ALWAYS"),
		PrSummaryComment:        types.StringValue("ALWAYS"),
		SkipCheckRuns:           types.StringValue("ALWAYS"),
		ConfigFileSupport:       types.StringValue("ENABLED"),
		PrSummaryAppendix:       types.StringValue("base appendix"),
		ArchiveConditions:       types.ListValueMust(types.StringType, []attr.Value{types.StringValue("AVOID_SCAN")}),
		UnavailableConditions:   types.ListNull(types.StringType),
	}
	overlay := &ConfigSettingsModel{
		DisableScanPullRequests: types.BoolNull(),
		CommentsOnPullRequests:  types.StringNull(),
		PrSummaryComment:        types.StringValue("ONLY_ON_FAILED_ISSUES"),
		SkipCheckRuns:           types.StringNull(),
		ConfigFileSupport:       types.StringNull(),
		PrSummaryAppendix:       types.StringNull(),
		ArchiveConditions:       types.ListNull(types.StringType),
		UnavailableConditions:   types.ListNull(types.StringType),
	}

	merged := MergeConfigSettings(base, overlay)

	// overlay wins on the field it explicitly set.
	if merged.PrSummaryComment.ValueString() != "ONLY_ON_FAILED_ISSUES" {
		t.Fatalf("expected overlay to win on PrSummaryComment, got: %v", merged.PrSummaryComment)
	}

	// base is preserved for every field overlay left null/unknown, and the
	// result is a COMPLETE object (nothing null) so the API's required-field
	// check is satisfied.
	if merged.DisableScanPullRequests.ValueBool() != base.DisableScanPullRequests.ValueBool() {
		t.Fatalf("expected base DisableScanPullRequests kept, got: %v", merged.DisableScanPullRequests)
	}
	if !merged.CommentsOnPullRequests.Equal(base.CommentsOnPullRequests) {
		t.Fatalf("expected base CommentsOnPullRequests kept, got: %v", merged.CommentsOnPullRequests)
	}
	if !merged.SkipCheckRuns.Equal(base.SkipCheckRuns) {
		t.Fatalf("expected base SkipCheckRuns kept, got: %v", merged.SkipCheckRuns)
	}
	if !merged.ConfigFileSupport.Equal(base.ConfigFileSupport) {
		t.Fatalf("expected base ConfigFileSupport kept, got: %v", merged.ConfigFileSupport)
	}
	if !merged.PrSummaryAppendix.Equal(base.PrSummaryAppendix) {
		t.Fatalf("expected base PrSummaryAppendix kept, got: %v", merged.PrSummaryAppendix)
	}
	if !merged.ArchiveConditions.Equal(base.ArchiveConditions) {
		t.Fatalf("expected base ArchiveConditions kept, got: %v", merged.ArchiveConditions)
	}

	if merged.CommentsOnPullRequests.IsNull() || merged.PrSummaryComment.IsNull() ||
		merged.SkipCheckRuns.IsNull() || merged.ConfigFileSupport.IsNull() {
		t.Fatalf("merged result must be complete (no required field left null): %+v", merged)
	}
}

// TestMergeConfigSettings_OverlaySetFieldsAllOverrideBase confirms every
// overlay-settable field is wired into the merge, not just PrSummaryComment.
func TestMergeConfigSettings_OverlaySetFieldsAllOverrideBase(t *testing.T) {
	base := ConfigSettingsModel{
		DisableScanPullRequests: types.BoolValue(false),
		CommentsOnPullRequests:  types.StringValue("ALWAYS"),
		PrSummaryComment:        types.StringValue("ALWAYS"),
		SkipCheckRuns:           types.StringValue("ALWAYS"),
		ConfigFileSupport:       types.StringValue("ENABLED"),
		PrSummaryAppendix:       types.StringValue("base"),
		ArchiveConditions:       types.ListNull(types.StringType),
		UnavailableConditions:   types.ListNull(types.StringType),
	}
	overlay := &ConfigSettingsModel{
		DisableScanPullRequests: types.BoolValue(true),
		CommentsOnPullRequests:  types.StringValue("NEVER"),
		PrSummaryComment:        types.StringValue("ONLY_ON_FAILED_ISSUES"),
		SkipCheckRuns:           types.StringValue("ONLY_ON_INTERNAL_ISSUE"),
		ConfigFileSupport:       types.StringValue("DISABLED"),
		PrSummaryAppendix:       types.StringValue("overlay"),
		ArchiveConditions:       types.ListValueMust(types.StringType, []attr.Value{types.StringValue("DELETE_REPO")}),
		UnavailableConditions:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("DELETE_REPO")}),
	}

	merged := MergeConfigSettings(base, overlay)

	if !merged.DisableScanPullRequests.Equal(overlay.DisableScanPullRequests) {
		t.Fatalf("expected overlay DisableScanPullRequests, got: %v", merged.DisableScanPullRequests)
	}
	if !merged.CommentsOnPullRequests.Equal(overlay.CommentsOnPullRequests) {
		t.Fatalf("expected overlay CommentsOnPullRequests, got: %v", merged.CommentsOnPullRequests)
	}
	if !merged.PrSummaryComment.Equal(overlay.PrSummaryComment) {
		t.Fatalf("expected overlay PrSummaryComment, got: %v", merged.PrSummaryComment)
	}
	if !merged.SkipCheckRuns.Equal(overlay.SkipCheckRuns) {
		t.Fatalf("expected overlay SkipCheckRuns, got: %v", merged.SkipCheckRuns)
	}
	if !merged.ConfigFileSupport.Equal(overlay.ConfigFileSupport) {
		t.Fatalf("expected overlay ConfigFileSupport, got: %v", merged.ConfigFileSupport)
	}
	if !merged.PrSummaryAppendix.Equal(overlay.PrSummaryAppendix) {
		t.Fatalf("expected overlay PrSummaryAppendix, got: %v", merged.PrSummaryAppendix)
	}
	if !merged.ArchiveConditions.Equal(overlay.ArchiveConditions) {
		t.Fatalf("expected overlay ArchiveConditions, got: %v", merged.ArchiveConditions)
	}
	if !merged.UnavailableConditions.Equal(overlay.UnavailableConditions) {
		t.Fatalf("expected overlay UnavailableConditions, got: %v", merged.UnavailableConditions)
	}
}

// TestMergeConfigSettings_UnknownOverlayFieldsDoNotOverrideBase guards
// against treating an unknown (not-yet-planned) value as an explicit user
// override; only non-null, non-unknown overlay fields should win.
func TestMergeConfigSettings_UnknownOverlayFieldsDoNotOverrideBase(t *testing.T) {
	base := ConfigSettingsModel{
		PrSummaryComment:  types.StringValue("ALWAYS"),
		ArchiveConditions: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("AVOID_SCAN")}),
	}
	overlay := &ConfigSettingsModel{
		PrSummaryComment:  types.StringUnknown(),
		ArchiveConditions: types.ListUnknown(types.StringType),
	}

	merged := MergeConfigSettings(base, overlay)

	if !merged.PrSummaryComment.Equal(base.PrSummaryComment) {
		t.Fatalf("expected base kept for unknown overlay PrSummaryComment, got: %v", merged.PrSummaryComment)
	}
	if !merged.ArchiveConditions.Equal(base.ArchiveConditions) {
		t.Fatalf("expected base kept for unknown overlay ArchiveConditions, got: %v", merged.ArchiveConditions)
	}
}
