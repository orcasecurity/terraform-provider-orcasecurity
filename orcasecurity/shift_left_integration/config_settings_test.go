package shift_left_integration

import (
	"context"
	"encoding/json"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// validateConfigSetting runs every validator declared on a configuration_settings
// attribute, the way the framework would during plan.
func validateConfigSetting(t *testing.T, attrName, value string) diag.Diagnostics {
	t.Helper()
	attr, ok := ConfigSettingsAttributes()[attrName].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s must be a StringAttribute", attrName)
	}
	var diags diag.Diagnostics
	for _, v := range attr.Validators {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("configuration_settings").AtName(attrName),
			ConfigValue: types.StringValue(value),
		}, resp)
		diags.Append(resp.Diagnostics...)
	}
	return diags
}

// The account/group PUT types skip_check_runs as the three-value
// PerformActionStatus for every provider. GitLab groups used to be restricted to
// ALWAYS/NEVER here, which blocked a value the API accepts (verified live); the
// two-value enum is the GitLab repository-level contract only.
// The account PUT requires these four enums and rejects "", but the API returns
// "" on some legacy units. Read maps that to null, so without a write-path
// fallback an adopt/update of such a unit would send "" back and 400.
func TestExpandConfigSettings_NeverSendsEmptyRequiredEnums(t *testing.T) {
	// A unit whose live config has empty enums, round-tripped through Read.
	flattened := FlattenConfigSettings(api_client.ShiftLeftConfigSettings{})
	for name, got := range map[string]types.String{
		"comments_on_pull_requests": flattened.CommentsOnPullRequests,
		"pr_summary_comment":        flattened.PrSummaryComment,
		"skip_check_runs":           flattened.SkipCheckRuns,
		"config_file_support":       flattened.ConfigFileSupport,
	} {
		if !got.IsNull() {
			t.Fatalf("precondition: %s should flatten to null, got %v", name, got)
		}
	}

	expanded := ExpandConfigSettings(&flattened)
	for name, got := range map[string]string{
		"comments_on_pull_requests": expanded.CommentsOnPullRequests,
		"pr_summary_comment":        expanded.PrSummaryComment,
		"skip_check_runs":           expanded.SkipCheckRuns,
		"config_file_support":       expanded.ConfigFileSupport,
	} {
		if got == "" {
			t.Errorf("%s must not serialize empty; the API rejects it", name)
		}
	}

	// A nil model is the same hazard via a different door.
	nilExpanded := ExpandConfigSettings(nil)
	if nilExpanded.CommentsOnPullRequests == "" || nilExpanded.SkipCheckRuns == "" ||
		nilExpanded.PrSummaryComment == "" || nilExpanded.ConfigFileSupport == "" {
		t.Errorf("nil config must still send valid enums, got %+v", nilExpanded)
	}
}

// Live values must win over the fallback: defaulting may only fill blanks.
func TestExpandConfigSettings_PreservesNonEmptyEnums(t *testing.T) {
	m := &ConfigSettingsModel{
		PRSettingsModel: shift_left_common.PRSettingsModel{
			CommentsOnPullRequests: types.StringValue("NEVER"),
			PrSummaryComment:       types.StringValue("ONLY_ON_FAILED_ISSUES"),
			SkipCheckRuns:          types.StringValue("ONLY_ON_INTERNAL_ISSUE"),
			ConfigFileSupport:      types.StringValue("DISABLED"),
		},
	}
	got := ExpandConfigSettings(m)
	if got.CommentsOnPullRequests != "NEVER" || got.PrSummaryComment != "ONLY_ON_FAILED_ISSUES" ||
		got.SkipCheckRuns != "ONLY_ON_INTERNAL_ISSUE" || got.ConfigFileSupport != "DISABLED" {
		t.Errorf("fallback overwrote configured enums: %+v", got)
	}
}

// pr_summary_appendix is optional server-side and "" means "clear", so it must
// not be swept up by the required-enum fallback.
func TestExpandConfigSettings_AppendixEmptyStaysEmpty(t *testing.T) {
	if got := ExpandConfigSettings(&ConfigSettingsModel{}).PrSummaryAppendix; got != "" {
		t.Errorf("pr_summary_appendix must stay empty to clear, got %q", got)
	}
}

func TestConfigSettingsAttributes_SkipCheckRunsAcceptsAllThreeValues(t *testing.T) {
	for _, value := range []string{"ALWAYS", "NEVER", "ONLY_ON_INTERNAL_ISSUE"} {
		if d := validateConfigSetting(t, "skip_check_runs", value); d.HasError() {
			t.Errorf("skip_check_runs must accept %q: %v", value, d)
		}
	}
	if d := validateConfigSetting(t, "skip_check_runs", "SOMETIMES"); !d.HasError() {
		t.Error("skip_check_runs must reject a value outside the enum")
	}
}

func TestConfigSettingsRoundTrip(t *testing.T) {
	m := &ConfigSettingsModel{
		PRSettingsModel: shift_left_common.PRSettingsModel{
			DisableScanPullRequests: types.BoolValue(false),
			CommentsOnPullRequests:  types.StringValue("ONLY_ON_FAILED_ISSUES"),
			PrSummaryComment:        types.StringValue("ONLY_ON_FAILED_ISSUES"),
			ConfigFileSupport:       types.StringValue("ENABLED"),
		},
		PrSummaryAppendix: types.StringValue("note"),
		ArchiveConditions: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("AVOID_SCAN")}),
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
		PRSettingsModel: shift_left_common.PRSettingsModel{
			DisableScanPullRequests: types.BoolValue(true),
			ConfigFileSupport:       types.StringValue("DISABLED"),
		},
	}
	api := ExpandConfigSettings(m)
	if api.InstallationReposConfig != nil {
		t.Fatalf("expected nil InstallationReposConfig when no conditions set, got: %+v", api.InstallationReposConfig)
	}
}

func TestExpandConfigSettings_ExplicitEmptyListsClearsReposConfig(t *testing.T) {
	m := &ConfigSettingsModel{
		ArchiveConditions:     types.SetValueMust(types.StringType, []attr.Value{}),
		UnavailableConditions: types.SetValueMust(types.StringType, []attr.Value{}),
	}
	api := ExpandConfigSettings(m)
	if api.InstallationReposConfig == nil {
		t.Fatal("expected empty InstallationReposConfig object to clear server-side, got nil (omitted)")
	}
	// Both action sets must be present with an explicit empty conditions array. Omitting a
	// key leaves the previous conditions in place, so a nil action set would not clear.
	for name, actions := range map[string]*api_client.ShiftLeftArchiveActions{
		"archive_actions":     api.InstallationReposConfig.ArchiveActions,
		"unavailable_actions": api.InstallationReposConfig.UnavailableActions,
	} {
		if actions == nil {
			t.Fatalf("%s must be sent on clear, not omitted", name)
		}
		if len(actions.Conditions) != 0 {
			t.Fatalf("%s should clear to an empty list, got: %v", name, actions.Conditions)
		}
	}
}

// A known-empty archive list alongside a populated unavailable list must still clear the
// archive conditions: previously the empty side was omitted and silently kept server-side.
func TestExpandConfigSettings_AsymmetricClearStillSendsBothActions(t *testing.T) {
	m := &ConfigSettingsModel{
		ArchiveConditions:     types.SetValueMust(types.StringType, []attr.Value{}),
		UnavailableConditions: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("DELETE_REPO")}),
	}
	api := ExpandConfigSettings(m)
	if api.InstallationReposConfig == nil {
		t.Fatal("expected installation_repositories_configuration to be sent")
	}
	archive := api.InstallationReposConfig.ArchiveActions
	if archive == nil {
		t.Fatal("archive_actions omitted; the empty archive list would not clear server-side")
	}
	if len(archive.Conditions) != 0 {
		t.Fatalf("expected archive conditions cleared, got: %v", archive.Conditions)
	}
	unavailable := api.InstallationReposConfig.UnavailableActions
	if unavailable == nil || len(unavailable.Conditions) != 1 || unavailable.Conditions[0] != "DELETE_REPO" {
		t.Fatalf("expected unavailable conditions preserved, got: %+v", unavailable)
	}
}

// conditions must serialize as [] rather than being dropped by omitempty.
func TestExpandConfigSettings_EmptyConditionsSerializeAsArray(t *testing.T) {
	m := &ConfigSettingsModel{
		ArchiveConditions:     types.SetValueMust(types.StringType, []attr.Value{}),
		UnavailableConditions: types.SetValueMust(types.StringType, []attr.Value{}),
	}
	raw, err := json.Marshal(ExpandConfigSettings(m).InstallationReposConfig)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"archive_actions":{"conditions":[]},"unavailable_actions":{"conditions":[]}}`
	if string(raw) != want {
		t.Fatalf("wire format\n got: %s\nwant: %s", raw, want)
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
		ArchiveConditions:     types.SetValueMust(types.StringType, []attr.Value{}),
		UnavailableConditions: types.SetValueMust(types.StringType, []attr.Value{}),
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
		UnavailableConditions: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("DELETE_REPO")}),
	}
	api := ExpandConfigSettings(m)
	if api.InstallationReposConfig == nil {
		t.Fatalf("expected installation_repositories_configuration to be sent: %+v", api.InstallationReposConfig)
	}
	// archive_actions is sent as an explicit empty list. Callers reach Expand through Adopt,
	// which hydrates an unset archive list from the live unit first, so this cannot clobber.
	if archive := api.InstallationReposConfig.ArchiveActions; archive == nil || len(archive.Conditions) != 0 {
		t.Fatalf("expected archive_actions sent with empty conditions, got: %+v", archive)
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
	for _, key := range []string{"disable_scan_pull_requests", "comments_on_pull_requests", "pr_summary_comment", "skip_check_runs", "config_file_support", "pr_summary_appendix", "archive_conditions", "unavailable_conditions"} {
		if _, ok := attrs[key]; !ok {
			t.Fatalf("expected field %q to always be present", key)
		}
	}

	for _, k := range []string{"archive_conditions", "unavailable_conditions"} {
		s, ok := attrs[k].(schema.SetAttribute)
		if !ok || !s.Optional || !s.Computed {
			t.Fatalf("%s must be an Optional+Computed SetAttribute, got: %+v", k, attrs[k])
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

func TestMergeConfigSettings_NilOverlayReturnsBase(t *testing.T) {
	base := ConfigSettingsModel{
		PRSettingsModel: shift_left_common.PRSettingsModel{
			DisableScanPullRequests: types.BoolValue(true),
			CommentsOnPullRequests:  types.StringValue("ALWAYS"),
			PrSummaryComment:        types.StringValue("NEVER"),
			SkipCheckRuns:           types.StringValue("ALWAYS"),
			ConfigFileSupport:       types.StringValue("ENABLED"),
		},
	}
	merged := MergeConfigSettings(base, nil)
	if !merged.CommentsOnPullRequests.Equal(base.CommentsOnPullRequests) {
		t.Fatalf("expected base returned unchanged, got: %+v", merged)
	}
}

func TestMergeConfigSettings_PartialOverlayWinsOnSetFieldsOnly(t *testing.T) {
	base := ConfigSettingsModel{
		PRSettingsModel: shift_left_common.PRSettingsModel{
			DisableScanPullRequests: types.BoolValue(false),
			CommentsOnPullRequests:  types.StringValue("ALWAYS"),
			PrSummaryComment:        types.StringValue("ALWAYS"),
			SkipCheckRuns:           types.StringValue("ALWAYS"),
			ConfigFileSupport:       types.StringValue("ENABLED"),
		},
		PrSummaryAppendix:     types.StringValue("base appendix"),
		ArchiveConditions:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("AVOID_SCAN")}),
		UnavailableConditions: types.SetNull(types.StringType),
	}
	overlay := &ConfigSettingsModel{
		PRSettingsModel: shift_left_common.PRSettingsModel{
			DisableScanPullRequests: types.BoolNull(),
			CommentsOnPullRequests:  types.StringNull(),
			PrSummaryComment:        types.StringValue("ONLY_ON_FAILED_ISSUES"),
			SkipCheckRuns:           types.StringNull(),
			ConfigFileSupport:       types.StringNull(),
		},
		PrSummaryAppendix:     types.StringNull(),
		ArchiveConditions:     types.SetNull(types.StringType),
		UnavailableConditions: types.SetNull(types.StringType),
	}

	merged := MergeConfigSettings(base, overlay)

	if merged.PrSummaryComment.ValueString() != "ONLY_ON_FAILED_ISSUES" {
		t.Fatalf("expected overlay to win on PrSummaryComment, got: %v", merged.PrSummaryComment)
	}

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

func TestMergeConfigSettings_OverlaySetFieldsAllOverrideBase(t *testing.T) {
	base := ConfigSettingsModel{
		PRSettingsModel: shift_left_common.PRSettingsModel{
			DisableScanPullRequests: types.BoolValue(false),
			CommentsOnPullRequests:  types.StringValue("ALWAYS"),
			PrSummaryComment:        types.StringValue("ALWAYS"),
			SkipCheckRuns:           types.StringValue("ALWAYS"),
			ConfigFileSupport:       types.StringValue("ENABLED"),
		},
		PrSummaryAppendix:     types.StringValue("base"),
		ArchiveConditions:     types.SetNull(types.StringType),
		UnavailableConditions: types.SetNull(types.StringType),
	}
	overlay := &ConfigSettingsModel{
		PRSettingsModel: shift_left_common.PRSettingsModel{
			DisableScanPullRequests: types.BoolValue(true),
			CommentsOnPullRequests:  types.StringValue("NEVER"),
			PrSummaryComment:        types.StringValue("ONLY_ON_FAILED_ISSUES"),
			SkipCheckRuns:           types.StringValue("ONLY_ON_INTERNAL_ISSUE"),
			ConfigFileSupport:       types.StringValue("DISABLED"),
		},
		PrSummaryAppendix:     types.StringValue("overlay"),
		ArchiveConditions:     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("DELETE_REPO")}),
		UnavailableConditions: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("DELETE_REPO")}),
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

func TestMergeConfigSettings_UnknownOverlayFieldsDoNotOverrideBase(t *testing.T) {
	base := ConfigSettingsModel{
		PRSettingsModel:   shift_left_common.PRSettingsModel{PrSummaryComment: types.StringValue("ALWAYS")},
		ArchiveConditions: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("AVOID_SCAN")}),
	}
	overlay := &ConfigSettingsModel{
		PRSettingsModel:   shift_left_common.PRSettingsModel{PrSummaryComment: types.StringUnknown()},
		ArchiveConditions: types.SetUnknown(types.StringType),
	}

	merged := MergeConfigSettings(base, overlay)

	if !merged.PrSummaryComment.Equal(base.PrSummaryComment) {
		t.Fatalf("expected base kept for unknown overlay PrSummaryComment, got: %v", merged.PrSummaryComment)
	}
	if !merged.ArchiveConditions.Equal(base.ArchiveConditions) {
		t.Fatalf("expected base kept for unknown overlay ArchiveConditions, got: %v", merged.ArchiveConditions)
	}
}
