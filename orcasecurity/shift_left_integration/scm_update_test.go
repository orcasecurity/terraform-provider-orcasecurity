package shift_left_integration

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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

// SCAN_ALL remap: API rejects legacy mode on update.
func TestAdopt_RemapsLegacyScanAllMode(t *testing.T) {
	ad := Adopt(
		types.StringNull(), // installation_mode unset in config
		types.BoolNull(),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{},
		ExistingUnit{InstallationMode: "SCAN_ALL", PolicyIDs: []string{"pol-1"}},
	)
	if ad.Body.InstallationMode != "SELECTED_REPOSITORIES" {
		t.Fatalf("expected legacy SCAN_ALL remapped to SELECTED_REPOSITORIES, got %q", ad.Body.InstallationMode)
	}
	// An explicit user-set mode is never remapped (the schema validator already
	// rejects SCAN_ALL in config).
	ad = Adopt(
		types.StringValue("SCAN_ALL_INCLUDE_FUTURE"),
		types.BoolNull(),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{},
		ExistingUnit{InstallationMode: "SCAN_ALL"},
	)
	if ad.Body.InstallationMode != "SCAN_ALL_INCLUDE_FUTURE" {
		t.Fatalf("expected user mode kept, got %q", ad.Body.InstallationMode)
	}
}

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
