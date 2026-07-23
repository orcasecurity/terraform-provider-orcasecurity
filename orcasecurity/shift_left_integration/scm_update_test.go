package shift_left_integration

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestAdopt_HydratesPoliciesFromExisting asserts that when the user leaves
// policies_ids unset, Adopt sends the unit's existing explicit policies rather
// than wiping them (regression guard for the adopt-existing policies wipe).
func TestAdopt_HydratesPoliciesFromExisting(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetNull(types.StringType), // policies_ids unset
		nil,
		ProjectIntent{},
		ExistingUnit{
			InstallationMode: "SCAN_ALL_INCLUDE_FUTURE",
			DefaultPolicies:  false,
			PolicyIDs:        []string{"pol-1", "pol-2"},
		},
	)
	if len(ad.Body.Policies) != 2 {
		t.Fatalf("expected existing policies preserved, got %v", ad.Body.Policies)
	}
	if ad.InstallationMode.ValueString() != "SCAN_ALL_INCLUDE_FUTURE" {
		t.Errorf("expected installation_mode hydrated from existing, got %q", ad.InstallationMode.ValueString())
	}
}

// TestAdopt_UserPoliciesWin asserts explicit policies_ids override the existing ones.
func TestAdopt_UserPoliciesWin(t *testing.T) {
	ad := Adopt(
		types.StringValue("SCAN_ALL"),
		types.BoolValue(false),
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-9")}),
		nil,
		ProjectIntent{},
		ExistingUnit{DefaultPolicies: false, PolicyIDs: []string{"pol-1", "pol-2"}},
	)
	if len(ad.Body.Policies) != 1 || ad.Body.Policies[0] != "pol-9" {
		t.Fatalf("expected user policies to win, got %v", ad.Body.Policies)
	}
}

// TestAdopt_DefaultPoliciesClearsPolicies asserts default_policies=true sends [].
func TestAdopt_DefaultPoliciesClearsPolicies(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(true),
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-1")}),
		nil,
		ProjectIntent{},
		ExistingUnit{DefaultPolicies: false, PolicyIDs: []string{"pol-1", "pol-2"}},
	)
	if len(ad.Body.Policies) != 0 {
		t.Fatalf("expected empty policies when default_policies=true, got %v", ad.Body.Policies)
	}
	if !ad.Body.DefaultPolicies {
		t.Error("expected default_policies=true in body")
	}
}

// TestAdopt_PreservesProject asserts a project-bound unit keeps its scan-all
// project (project_id echoed, policies dropped, matching the UI's XOR write).
func TestAdopt_PreservesProject(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{},
		ExistingUnit{
			DefaultPolicies: false,
			PolicyIDs:       []string{"pol-1"},
			ProjectID:       "proj-1",
		},
	)
	if ad.Body.ProjectID != "proj-1" {
		t.Fatalf("expected project_id preserved, got %q", ad.Body.ProjectID)
	}
	if ad.Body.Policies != nil {
		t.Fatalf("expected policies dropped when project-bound, got %v", ad.Body.Policies)
	}
}

// TestAdopt_BindsProjectFromConfig asserts a config project_id binds the unit
// to that project (policies dropped) even when the unit had none before.
func TestAdopt_BindsProjectFromConfig(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{FromConfig: types.StringValue("proj-new")},
		ExistingUnit{PolicyIDs: []string{"pol-1"}},
	)
	if ad.Body.ProjectID != "proj-new" {
		t.Fatalf("expected project bound from config, got %q", ad.Body.ProjectID)
	}
	if ad.Body.Policies != nil {
		t.Fatalf("expected policies dropped when project-bound, got %v", ad.Body.Policies)
	}
}

// TestAdopt_ClearsProjectWhenConfigEmpty asserts project_id="" clears an
// existing binding and falls back to policies.
func TestAdopt_ClearsProjectWhenConfigEmpty(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{FromConfig: types.StringValue("")},
		ExistingUnit{PolicyIDs: []string{"pol-1"}, ProjectID: "proj-old"},
	)
	if ad.Body.ProjectID != "" {
		t.Fatalf("expected project cleared, got %q", ad.Body.ProjectID)
	}
	if len(ad.Body.Policies) != 1 || ad.Body.Policies[0] != "pol-1" {
		t.Fatalf("expected policies restored after clearing project, got %v", ad.Body.Policies)
	}
}

// TestAdopt_PoliciesIntentClearsProject asserts choosing policies in config
// clears an existing project binding (mirrors the UI's XOR).
func TestAdopt_PoliciesIntentClearsProject(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-9")}),
		nil,
		ProjectIntent{PoliciesIntent: true},
		ExistingUnit{ProjectID: "proj-old"},
	)
	if ad.Body.ProjectID != "" {
		t.Fatalf("expected project cleared by policies intent, got %q", ad.Body.ProjectID)
	}
	if len(ad.Body.Policies) != 1 || ad.Body.Policies[0] != "pol-9" {
		t.Fatalf("expected user policies applied, got %v", ad.Body.Policies)
	}
}

// TestAdopt_MergesConfigSettings asserts the user overlay merges on top of the
// live configuration_settings rather than replacing them wholesale.
func TestAdopt_MergesConfigSettings(t *testing.T) {
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
	if ad.Body.ConfigSettings.PrSummaryComment != "ONLY_ON_FAILED_ISSUES" {
		t.Errorf("expected overlay pr_summary_comment, got %q", ad.Body.ConfigSettings.PrSummaryComment)
	}
	if ad.Body.ConfigSettings.CommentsOnPullRequests != "ALWAYS" {
		t.Errorf("expected live comments_on_pull_requests preserved, got %q", ad.Body.ConfigSettings.CommentsOnPullRequests)
	}
}
