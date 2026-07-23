package shift_left_gitlab_group_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGitlabGroup_import(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_GL_INSTALLATION_ID")
	orcaGroupID := os.Getenv("ORCA_TEST_GL_GROUP_ID")
	if installationID == "" || orcaGroupID == "" {
		t.Skip("ORCA_TEST_GL_INSTALLATION_ID / ORCA_TEST_GL_GROUP_ID not set")
	}

	orcasecurity.TestAccPreCheck(t)
	client := acctest.APIClient(t)
	original, err := client.GetGitlabGroup(installationID, orcaGroupID)
	if err != nil {
		t.Fatalf("failed to snapshot gitlab group %s/%s: %s", installationID, orcaGroupID, err)
	}
	if original == nil {
		t.Skipf("gitlab group %s/%s not found; cannot run adopt test", installationID, orcaGroupID)
	}
	gitlabGroupID := original.GitlabGroupID
	t.Cleanup(func() {
		restoreGitlabGroup(t, client, installationID, gitlabGroupID, original)
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_gitlab_group" "t" {
  installation_id  = %q
  gitlab_group_id  = %d
  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}`, installationID, gitlabGroupID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_gitlab_group.t", "installation_id", installationID),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_gitlab_group.t", "gitlab_group_id", fmt.Sprintf("%d", gitlabGroupID)),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_gitlab_group.t", "configuration_settings.pr_summary_comment", "ONLY_ON_FAILED_ISSUES"),
				),
			},
			{
				ResourceName:      "orcasecurity_shift_left_gitlab_group.t",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s/%d", installationID, gitlabGroupID), nil
				},
			},
		},
	})
}

// restoreGitlabGroup re-integrates the unit if destroy removed it, then restores config.
func restoreGitlabGroup(t *testing.T, client *api_client.APIClient, installationID string, gitlabGroupID int64, original *api_client.GitlabGroup) {
	t.Helper()
	body := acctest.RestoreScmBody(original.InstallationMode, original.DefaultPolicies, original.Policies, original.Project, original.ConfigSettings)
	cur, err := client.FindGitlabGroupByGitlabID(installationID, gitlabGroupID)
	if err != nil {
		t.Errorf("restore lookup: %s", err)
		return
	}
	if cur == nil {
		if err := client.IntegrateGitlabUnit(api_client.GitlabUnitIntegrate{
			InstallationID: installationID,
			GitlabGroupID:  gitlabGroupID,
			Body:           body,
		}); err != nil {
			t.Errorf("failed to re-integrate gitlab group %d: %s", gitlabGroupID, err)
		}
		return
	}
	if _, err := client.UpdateGitlabGroup(installationID, cur.ID, body); err != nil {
		t.Errorf("failed to restore gitlab group %s: %s", cur.ID, err)
	}
}
