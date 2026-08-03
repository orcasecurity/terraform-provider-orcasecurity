package shift_left_integration

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func policyRefs(ids ...string) []api_client.ScmPolicyRef {
	refs := make([]api_client.ScmPolicyRef, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, api_client.ScmPolicyRef{ID: id})
	}
	return refs
}

func projectRef(id string) *api_client.ScmProjectRef { return &api_client.ScmProjectRef{ID: id} }

func TestAdopt_HydratesPoliciesFromExisting(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetNull(types.StringType), // policies_ids unset
		nil,
		ProjectIntent{},
		api_client.ScmUnitCommonFields{
			InstallationMode: "SCAN_ALL_INCLUDE_FUTURE",
			DefaultPolicies:  false,
			Policies:         policyRefs("pol-1", "pol-2"),
		},
	)
	if len(ad.Policies) != 2 {
		t.Fatalf("expected existing policies preserved, got %v", ad.Policies)
	}
	if ad.InstallationMode != "SCAN_ALL_INCLUDE_FUTURE" {
		t.Errorf("expected installation_mode hydrated from existing, got %q", ad.InstallationMode)
	}
}

func TestAdopt_UserPoliciesWin(t *testing.T) {
	ad := Adopt(
		types.StringValue("SCAN_ALL"),
		types.BoolValue(false),
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-9")}),
		nil,
		ProjectIntent{PoliciesIntent: true},
		api_client.ScmUnitCommonFields{DefaultPolicies: false, Policies: policyRefs("pol-1", "pol-2")},
	)
	if len(ad.Policies) != 1 || ad.Policies[0] != "pol-9" {
		t.Fatalf("expected user policies to win, got %v", ad.Policies)
	}
}

// Stale default_policies=true carried via UseStateForUnknown must not discard
// an explicit policies_ids list on the wire.
func TestAdopt_PoliciesIntentForcesDefaultPoliciesFalse(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(true), // would silently clear policies without the force
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-9")}),
		nil,
		ProjectIntent{PoliciesIntent: true},
		api_client.ScmUnitCommonFields{DefaultPolicies: true},
	)
	if ad.DefaultPolicies {
		t.Error("expected default_policies forced false when policies_ids are explicit")
	}
	if len(ad.Policies) != 1 || ad.Policies[0] != "pol-9" {
		t.Fatalf("expected policies preserved on the wire, got %v", ad.Policies)
	}
}

func TestAdopt_DefaultPoliciesClearsPolicies(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(true),
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-1")}),
		nil,
		ProjectIntent{},
		api_client.ScmUnitCommonFields{DefaultPolicies: false, Policies: policyRefs("pol-1", "pol-2")},
	)
	if len(ad.Policies) != 0 {
		t.Fatalf("expected empty policies when default_policies=true, got %v", ad.Policies)
	}
	if !ad.DefaultPolicies {
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
		api_client.ScmUnitCommonFields{
			DefaultPolicies: false,
			Policies:        policyRefs("pol-1"),
			Project:         projectRef("proj-1"),
		},
	)
	if ad.ProjectID != "proj-1" {
		t.Fatalf("expected project_id preserved, got %q", ad.ProjectID)
	}
	if ad.Policies != nil {
		t.Fatalf("expected policies dropped when project-bound, got %v", ad.Policies)
	}
}

func TestAdopt_BindsProjectFromConfig(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{FromConfig: types.StringValue("proj-new")},
		api_client.ScmUnitCommonFields{Policies: policyRefs("pol-1")},
	)
	if ad.ProjectID != "proj-new" {
		t.Fatalf("expected project bound from config, got %q", ad.ProjectID)
	}
	if ad.Policies != nil {
		t.Fatalf("expected policies dropped when project-bound, got %v", ad.Policies)
	}
}

func TestAdopt_ClearsProjectWhenConfigEmpty(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{FromConfig: types.StringValue("")},
		api_client.ScmUnitCommonFields{Policies: policyRefs("pol-1"), Project: projectRef("proj-old")},
	)
	if ad.ProjectID != "" {
		t.Fatalf("expected project cleared, got %q", ad.ProjectID)
	}
	if len(ad.Policies) != 1 || ad.Policies[0] != "pol-1" {
		t.Fatalf("expected policies restored after clearing project, got %v", ad.Policies)
	}
}

func TestAdopt_PoliciesIntentClearsProject(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-9")}),
		nil,
		ProjectIntent{PoliciesIntent: true},
		api_client.ScmUnitCommonFields{Project: projectRef("proj-old")},
	)
	if ad.ProjectID != "" {
		t.Fatalf("expected project cleared by policies intent, got %q", ad.ProjectID)
	}
	if len(ad.Policies) != 1 || ad.Policies[0] != "pol-9" {
		t.Fatalf("expected user policies applied, got %v", ad.Policies)
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
		api_client.ScmUnitCommonFields{InstallationMode: "SCAN_ALL", Policies: policyRefs("pol-1")},
	)
	if ad.InstallationMode != "SELECTED_REPOSITORIES" {
		t.Fatalf("expected legacy SCAN_ALL remapped to SELECTED_REPOSITORIES, got %q", ad.InstallationMode)
	}
	// An explicit user-set mode is never remapped (the schema validator already
	// rejects SCAN_ALL in config).
	ad = Adopt(
		types.StringValue("SCAN_ALL_INCLUDE_FUTURE"),
		types.BoolNull(),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{},
		api_client.ScmUnitCommonFields{InstallationMode: "SCAN_ALL"},
	)
	if ad.InstallationMode != "SCAN_ALL_INCLUDE_FUTURE" {
		t.Fatalf("expected user mode kept, got %q", ad.InstallationMode)
	}
}

// default_policies is orthogonal to project_id: it must not clear a bound project.
func TestAdopt_DefaultPoliciesKeepsProject(t *testing.T) {
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(true),           // default_policies=true
		types.SetNull(types.StringType), // no explicit policies list
		nil,
		ProjectIntent{}, // no explicit-policies intent, project_id omitted in config
		api_client.ScmUnitCommonFields{Project: projectRef("proj-1"), Policies: policyRefs("pol-1")},
	)
	if ad.ProjectID != "proj-1" {
		t.Fatalf("expected project preserved with default_policies, got %q", ad.ProjectID)
	}
	if !ad.DefaultPolicies {
		t.Error("expected default_policies=true sent alongside project_id")
	}
}

func TestProjectIntentFrom_OnlyExplicitPoliciesCount(t *testing.T) {
	pi := ProjectIntentFrom(types.StringNull(), types.SetNull(types.StringType))
	if pi.PoliciesIntent {
		t.Error("default_policies alone must not count as explicit-policies intent")
	}
	pi = ProjectIntentFrom(types.StringNull(),
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("p")}))
	if !pi.PoliciesIntent {
		t.Error("explicit policies_ids must set PoliciesIntent")
	}
}

func TestAdopt_MergesConfigSettings(t *testing.T) {
	overlay := &ConfigSettingsModel{
		PRSettingsModel: shift_left_common.PRSettingsModel{PrSummaryComment: types.StringValue("ONLY_ON_FAILED_ISSUES")},
	}
	ad := Adopt(
		types.StringNull(),
		types.BoolValue(false),
		types.SetNull(types.StringType),
		overlay,
		ProjectIntent{},
		api_client.ScmUnitCommonFields{
			ConfigSettings: api_client.ShiftLeftConfigSettings{
				CommentsOnPullRequests: "ALWAYS",
				PrSummaryComment:       "ALWAYS",
				SkipCheckRuns:          "ALWAYS",
				ConfigFileSupport:      "ENABLED",
			},
		},
	)
	if ad.ConfigSettings.PrSummaryComment != "ONLY_ON_FAILED_ISSUES" {
		t.Errorf("expected overlay pr_summary_comment, got %q", ad.ConfigSettings.PrSummaryComment)
	}
	if ad.ConfigSettings.CommentsOnPullRequests != "ALWAYS" {
		t.Errorf("expected live comments_on_pull_requests preserved, got %q", ad.ConfigSettings.CommentsOnPullRequests)
	}
}
