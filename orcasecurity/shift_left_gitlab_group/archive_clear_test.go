package shift_left_gitlab_group_test

import (
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestAccGitlabGroup_clearsArchiveConditions verifies that an explicit empty
// archive_conditions/unavailable_conditions overlay clears
// installation_repositories_configuration on the live unit (finding #3).
//
// Requires ORCA_TEST_GL_INSTALLATION_ID and ORCA_TEST_GL_GROUP_ID.
func TestAccGitlabGroup_clearsArchiveConditions(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_GL_INSTALLATION_ID")
	groupID := os.Getenv("ORCA_TEST_GL_GROUP_ID")
	if installationID == "" || groupID == "" {
		t.Skip("ORCA_TEST_GL_INSTALLATION_ID / ORCA_TEST_GL_GROUP_ID not set")
	}

	orcasecurity.TestAccPreCheck(t)
	client := orcasecurity.TestAPIClient(t)

	original, err := client.GetGitlabGroup(installationID, groupID)
	if err != nil {
		t.Fatalf("snapshot failed: %s", err)
	}
	if original == nil {
		t.Skipf("gitlab group %s/%s not found", installationID, groupID)
	}
	t.Cleanup(func() {
		if _, err := client.UpdateGitlabGroup(installationID, groupID, orcasecurity.RestoreScmBody(
			original.InstallationMode, original.DefaultPolicies, original.Policies, original.Project, original.ConfigSettings,
		)); err != nil {
			t.Errorf("restore failed for %s/%s: %s", installationID, groupID, err)
		}
	})

	// 1) Set archive + unavailable conditions.
	withConditions := original.ConfigSettings
	withConditions.InstallationReposConfig = &api_client.ShiftLeftInstallationReposConfig{
		ArchiveActions:     &api_client.ShiftLeftArchiveActions{Conditions: []string{"AVOID_SCAN", "DELETE_REPO"}},
		UnavailableActions: &api_client.ShiftLeftArchiveActions{Conditions: []string{"DELETE_REPO"}},
	}
	set, err := client.UpdateGitlabGroup(installationID, groupID, orcasecurity.RestoreScmBody(
		original.InstallationMode, original.DefaultPolicies, original.Policies, original.Project, withConditions,
	))
	if err != nil {
		t.Fatalf("set conditions failed: %s", err)
	}
	if set.ConfigSettings.InstallationReposConfig == nil {
		t.Fatal("precondition failed: expected installation_repositories_configuration after set")
	}

	// 2) Adopt with explicit empty condition lists (terraform clear intent).
	overlay := &shift_left_integration.ConfigSettingsModel{
		ArchiveConditions:     types.ListValueMust(types.StringType, []attr.Value{}),
		UnavailableConditions: types.ListValueMust(types.StringType, []attr.Value{}),
	}
	ad := shift_left_integration.Adopt(
		types.StringNull(), types.BoolNull(), types.SetNull(types.StringType), overlay,
		shift_left_integration.ProjectIntent{},
		shift_left_integration.ExistingFromAPI(set.InstallationMode, set.DefaultPolicies, set.Policies, set.Project, set.ConfigSettings),
	)
	if ad.Body.ConfigSettings.InstallationReposConfig == nil {
		t.Fatal("Expand must send explicit empty installation_repositories_configuration on clear")
	}

	cleared, err := client.UpdateGitlabGroup(installationID, groupID, ad.Body)
	if err != nil {
		t.Fatalf("clear update failed: %s", err)
	}
	// API may return null or an empty {} object; either is a successful clear.
	if repos := cleared.ConfigSettings.InstallationReposConfig; repos != nil {
		hasArchive := repos.ArchiveActions != nil && len(repos.ArchiveActions.Conditions) > 0
		hasUnavailable := repos.UnavailableActions != nil && len(repos.UnavailableActions.Conditions) > 0
		if hasArchive || hasUnavailable {
			t.Fatalf("expected installation_repositories_configuration cleared, got %+v", repos)
		}
	}
	t.Log("archive/unavailable conditions cleared via empty lists")
}
