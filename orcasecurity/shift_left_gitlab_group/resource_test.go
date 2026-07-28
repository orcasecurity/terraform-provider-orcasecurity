package shift_left_gitlab_group_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGitlabGroup_import(t *testing.T) {
	installationID, gitlabGroupIDEnv, orcaGroupID := requireGitlabGroupTestEnv(t)
	// resource.Test always destroys on teardown, and this resource adopts a
	// pre-existing group it did not create; require an explicit opt-in before
	// tearing down a shared-lab group.
	if os.Getenv("ORCA_TEST_GL_ALLOW_DESTROY") == "" {
		t.Skip("ORCA_TEST_GL_ALLOW_DESTROY not set; refuse to DELETE a shared lab GitLab group")
	}

	orcasecurity.TestAccPreCheck(t)
	client := acctest.APIClient(t)
	client.InvalidateScmListCache()

	original := fetchGitlabGroupForTest(t, client, installationID, gitlabGroupIDEnv, orcaGroupID)
	requireDisposableGroup(t, original) // even with opt-in, only tear down an empty group
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
