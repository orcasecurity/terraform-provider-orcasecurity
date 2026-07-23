package shift_left_gitlab_group_test

import (
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Requires ORCA_TEST_GL_INSTALLATION_ID, ORCA_TEST_GL_GROUP_ID, ORCA_TEST_GL_PROJECT_ID.
func TestAccGitlabGroup_preservesProject(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_GL_INSTALLATION_ID")
	groupID := os.Getenv("ORCA_TEST_GL_GROUP_ID")
	projectID := os.Getenv("ORCA_TEST_GL_PROJECT_ID")
	if installationID == "" || groupID == "" || projectID == "" {
		t.Skip("ORCA_TEST_GL_INSTALLATION_ID / ORCA_TEST_GL_GROUP_ID / ORCA_TEST_GL_PROJECT_ID not set")
	}

	orcasecurity.TestAccPreCheck(t)
	client := acctest.APIClient(t)

	original, err := client.GetGitlabGroup(installationID, groupID)
	if err != nil {
		t.Fatalf("snapshot failed: %s", err)
	}
	if original == nil {
		t.Skipf("gitlab group %s/%s not found", installationID, groupID)
	}
	// Always restore the group to exactly how we found it (policies + config,
	// no project) once the test finishes.
	t.Cleanup(func() {
		if _, err := client.UpdateGitlabGroup(installationID, groupID, acctest.RestoreScmBody(
			original.InstallationMode, original.DefaultPolicies, original.Policies, original.Project, original.ConfigSettings,
		)); err != nil {
			t.Errorf("restore failed for %s/%s: %s", installationID, groupID, err)
		}
	})

	// 1) Bind a project (project_id XOR policies, mirroring the UI).
	bound, err := client.UpdateGitlabGroup(installationID, groupID, acctest.RestoreScmBody(
		original.InstallationMode, original.DefaultPolicies, original.Policies,
		&api_client.ScmProjectRef{ID: projectID}, original.ConfigSettings,
	))
	if err != nil {
		t.Fatalf("bind project failed: %s", err)
	}
	if api_client.ProjectRefID(bound.Project) != projectID {
		t.Fatalf("precondition failed: expected group bound to project %q, got %q", projectID, api_client.ProjectRefID(bound.Project))
	}
	t.Logf("group bound to project %q", projectID)

	overlay := &shift_left_integration.ConfigSettingsModel{
		PrSummaryComment: types.StringValue("ONLY_ON_FAILED_ISSUES"),
	}
	ad := shift_left_integration.Adopt(
		types.StringNull(), types.BoolNull(), types.SetNull(types.StringType), overlay,
		shift_left_integration.ProjectIntent{},
		shift_left_integration.ExistingUnit{
			InstallationMode: bound.InstallationMode,
			DefaultPolicies:  bound.DefaultPolicies,
			PolicyIDs:        api_client.PolicyRefIDs(bound.Policies),
			ConfigSettings:   bound.ConfigSettings,
			ProjectID:        api_client.ProjectRefID(bound.Project),
		},
	)
	if ad.Body.ProjectID != projectID {
		t.Fatalf("Adopt dropped project from PUT body: got project_id=%q", ad.Body.ProjectID)
	}

	updated, err := client.UpdateGitlabGroup(installationID, groupID, ad.Body)
	if err != nil {
		t.Fatalf("adopt update failed: %s", err)
	}

	if got := api_client.ProjectRefID(updated.Project); got != projectID {
		t.Fatalf("#4 regression: project dropped after adopt update, got %q", got)
	}
	if updated.ConfigSettings.PrSummaryComment != "ONLY_ON_FAILED_ISSUES" {
		t.Fatalf("expected config overlay applied, got pr_summary_comment=%q", updated.ConfigSettings.PrSummaryComment)
	}
	t.Logf("project %q preserved through adopt update; pr_summary_comment=%q", projectID, updated.ConfigSettings.PrSummaryComment)
}
