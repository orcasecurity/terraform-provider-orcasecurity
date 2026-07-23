package shift_left_azure_devops_account_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAzureDevopsAccount_import(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_AZ_INSTALLATION_ID")
	accountID := os.Getenv("ORCA_TEST_AZ_ACCOUNT_ID")
	if installationID == "" || accountID == "" {
		t.Skip("ORCA_TEST_AZ_INSTALLATION_ID / ORCA_TEST_AZ_ACCOUNT_ID not set")
	}

	// Snapshot the live account and restore it after the test. Adopt-existing
	// units are not TF-owned (Delete is a no-op), so without this the applied
	// config would leak into the lab environment.
	orcasecurity.TestAccPreCheck(t)
	client := acctest.APIClient(t)
	original, err := client.GetAzureDevopsAccount(installationID, accountID)
	if err != nil {
		t.Fatalf("failed to snapshot azure devops account %s/%s: %s", installationID, accountID, err)
	}
	if original == nil {
		t.Skipf("azure devops account %s/%s not found; cannot run adopt test", installationID, accountID)
	}
	t.Cleanup(func() {
		if _, err := client.UpdateAzureDevopsAccount(installationID, accountID, acctest.RestoreScmBody(original.InstallationMode, original.DefaultPolicies, original.Policies, original.Project, original.ConfigSettings)); err != nil {
			t.Errorf("failed to restore azure devops account %s/%s to its original config: %s", installationID, accountID, err)
		}
	})

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
