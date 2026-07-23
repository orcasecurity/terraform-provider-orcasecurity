package shift_left_gitlab_group_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGitlabGroup_import(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_GL_INSTALLATION_ID")
	groupID := os.Getenv("ORCA_TEST_GL_GROUP_ID")
	if installationID == "" || groupID == "" {
		t.Skip("ORCA_TEST_GL_INSTALLATION_ID / ORCA_TEST_GL_GROUP_ID not set")
	}

	// Snapshot the live group and restore it after the test. Adopt-existing
	// units are not TF-owned (Delete is a no-op), so without this the applied
	// config would leak into the lab environment.
	orcasecurity.TestAccPreCheck(t)
	client := orcasecurity.TestAPIClient(t)
	original, err := client.GetGitlabGroup(installationID, groupID)
	if err != nil {
		t.Fatalf("failed to snapshot gitlab group %s/%s: %s", installationID, groupID, err)
	}
	if original == nil {
		t.Skipf("gitlab group %s/%s not found; cannot run adopt test", installationID, groupID)
	}
	t.Cleanup(func() {
		if _, err := client.UpdateGitlabGroup(installationID, groupID, orcasecurity.RestoreScmBody(original.InstallationMode, original.DefaultPolicies, original.Policies, original.Project, original.ConfigSettings)); err != nil {
			t.Errorf("failed to restore gitlab group %s/%s to its original config: %s", installationID, groupID, err)
		}
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_gitlab_group" "t" {
  installation_id = %q
  group_id        = %q
  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}`, installationID, groupID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_gitlab_group.t", "installation_id", installationID),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_gitlab_group.t", "group_id", groupID),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_gitlab_group.t", "configuration_settings.pr_summary_comment", "ONLY_ON_FAILED_ISSUES"),
				),
			},
			{
				ResourceName:      "orcasecurity_shift_left_gitlab_group.t",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s/%s", installationID, groupID), nil
				},
			},
		},
	})
}
