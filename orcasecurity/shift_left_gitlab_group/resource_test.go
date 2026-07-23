package shift_left_gitlab_group_test

import (
	"fmt"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGitlabGroup_import(t *testing.T) {
	installationID, gitlabGroupIDEnv, orcaGroupID := requireGitlabGroupTestEnv(t)

	orcasecurity.TestAccPreCheck(t)
	client := acctest.APIClient(t)
	client.InvalidateScmListCache()

	original := fetchGitlabGroupForTest(t, client, installationID, gitlabGroupIDEnv, orcaGroupID)
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
