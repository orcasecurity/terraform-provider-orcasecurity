package shift_left_gitlab_group_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGitlabGroup_import(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_GL_INSTALLATION_ID")
	// Prefer numeric gitlab_group_id (stable). Fall back to Orca UUID for older env setups.
	gitlabGroupIDEnv := os.Getenv("ORCA_TEST_GL_GITLAB_GROUP_ID")
	orcaGroupID := os.Getenv("ORCA_TEST_GL_GROUP_ID")
	if installationID == "" || (gitlabGroupIDEnv == "" && orcaGroupID == "") {
		t.Skip("ORCA_TEST_GL_INSTALLATION_ID and ORCA_TEST_GL_GITLAB_GROUP_ID (or ORCA_TEST_GL_GROUP_ID) not set")
	}

	orcasecurity.TestAccPreCheck(t)
	client := acctest.APIClient(t)
	client.InvalidateScmListCache()

	var original *api_client.GitlabGroup
	var err error
	if gitlabGroupIDEnv != "" {
		n, perr := strconv.ParseInt(gitlabGroupIDEnv, 10, 64)
		if perr != nil {
			t.Fatalf("ORCA_TEST_GL_GITLAB_GROUP_ID: %v", perr)
		}
		original, err = client.FindGitlabGroupByGitlabID(installationID, n)
	} else {
		original, err = client.GetGitlabGroup(installationID, orcaGroupID)
	}
	if err != nil {
		t.Fatalf("failed to snapshot gitlab group: %s", err)
	}
	if original == nil {
		t.Skip("gitlab group not found; cannot run adopt test")
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
	client.InvalidateScmListCache()
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
		client.InvalidateScmListCache()
		if err2 := client.IntegrateGitlabUnit(api_client.GitlabUnitIntegrate{
			InstallationID: installationID,
			GitlabGroupID:  gitlabGroupID,
			Body:           body,
		}); err2 != nil {
			t.Errorf("failed to restore gitlab group %s (update: %v; re-integrate: %v)", cur.ID, err, err2)
		}
	}
}
