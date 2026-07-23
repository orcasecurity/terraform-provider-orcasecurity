package shift_left_azure_devops_account_test

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

func TestAccAzureDevopsAccount_import(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_AZ_INSTALLATION_ID")
	orcaAccountID := os.Getenv("ORCA_TEST_AZ_ACCOUNT_ID")
	if installationID == "" || orcaAccountID == "" {
		t.Skip("ORCA_TEST_AZ_INSTALLATION_ID / ORCA_TEST_AZ_ACCOUNT_ID not set")
	}

	orcasecurity.TestAccPreCheck(t)
	client := acctest.APIClient(t)
	original, err := client.GetAzureDevopsAccount(installationID, orcaAccountID)
	if err != nil {
		t.Fatalf("failed to snapshot azure devops account %s/%s: %s", installationID, orcaAccountID, err)
	}
	if original == nil {
		t.Skipf("azure devops account %s/%s not found; cannot run adopt test", installationID, orcaAccountID)
	}
	accountName := original.AccountName
	t.Cleanup(func() {
		restoreAzureAccount(t, client, installationID, accountName, original)
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_azure_devops_account" "t" {
  installation_id = %q
  account_name    = %q
  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}`, installationID, accountName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_azure_devops_account.t", "installation_id", installationID),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_azure_devops_account.t", "account_name", accountName),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_azure_devops_account.t", "configuration_settings.pr_summary_comment", "ONLY_ON_FAILED_ISSUES"),
				),
			},
			{
				ResourceName:      "orcasecurity_shift_left_azure_devops_account.t",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s/%s", installationID, accountName), nil
				},
			},
		},
	})
}

func restoreAzureAccount(t *testing.T, client *api_client.APIClient, installationID, accountName string, original *api_client.AzureDevopsAccount) {
	t.Helper()
	body := acctest.RestoreScmBody(original.InstallationMode, original.DefaultPolicies, original.Policies, original.Project, original.ConfigSettings)
	cur, err := client.FindAzureDevopsAccountByName(installationID, accountName)
	if err != nil {
		t.Errorf("restore lookup: %s", err)
		return
	}
	if cur == nil {
		if err := client.IntegrateAzureDevopsUnit(api_client.AzureDevopsUnitIntegrate{
			InstallationID: installationID,
			AccountName:    accountName,
			Body:           body,
		}); err != nil {
			t.Errorf("failed to re-integrate azure account %q: %s", accountName, err)
		}
		return
	}
	if _, err := client.UpdateAzureDevopsAccount(installationID, cur.ID, body); err != nil {
		t.Errorf("failed to restore azure account %s: %s", cur.ID, err)
	}
}
