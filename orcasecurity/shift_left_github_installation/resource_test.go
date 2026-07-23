package shift_left_github_installation_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubInstallation_import(t *testing.T) {
	id := os.Getenv("ORCA_TEST_GH_INSTALLATION_ID")
	if id == "" {
		t.Skip("ORCA_TEST_GH_INSTALLATION_ID not set")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_github_installation" "t" {
  installation_id = %q
  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}`, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_github_installation.t", "installation_id", id),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_github_installation.t", "configuration_settings.pr_summary_comment", "ONLY_ON_FAILED_ISSUES"),
				),
			},
			{
				ResourceName:      "orcasecurity_shift_left_github_installation.t",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
