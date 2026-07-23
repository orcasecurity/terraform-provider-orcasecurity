package shift_left_azure_devops_account_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAzureDevopsAccount_import(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_AZ_INSTALLATION_ID")
	accountID := os.Getenv("ORCA_TEST_AZ_ACCOUNT_ID")
	if installationID == "" || accountID == "" {
		t.Skip("ORCA_TEST_AZ_INSTALLATION_ID / ORCA_TEST_AZ_ACCOUNT_ID not set")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_azure_devops_account" "t" {
  installation_id = %q
  account_id      = %q
  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}`, installationID, accountID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_azure_devops_account.t", "installation_id", installationID),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_azure_devops_account.t", "account_id", accountID),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_azure_devops_account.t", "configuration_settings.pr_summary_comment", "ONLY_ON_FAILED_ISSUES"),
				),
			},
			{
				ResourceName:      "orcasecurity_shift_left_azure_devops_account.t",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s/%s", installationID, accountID), nil
				},
			},
		},
	})
}
