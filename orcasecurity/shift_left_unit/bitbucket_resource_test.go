package shift_left_unit_test

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

func TestAccBitbucketAccount_import(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_BB_INSTALLATION_ID")
	accountSlug := os.Getenv("ORCA_TEST_BB_ACCOUNT_SLUG")
	orcaAccountID := os.Getenv("ORCA_TEST_BB_ORCA_ACCOUNT_ID")
	if installationID == "" || (accountSlug == "" && orcaAccountID == "") {
		t.Skip("ORCA_TEST_BB_INSTALLATION_ID and ORCA_TEST_BB_ACCOUNT_SLUG (or ORCA_TEST_BB_ORCA_ACCOUNT_ID) not set")
	}

	client := acctest.APIClient(t)

	var original *api_client.BitbucketAccount
	var err error
	if accountSlug != "" {
		original, err = client.FindBitbucketAccountBySlug(installationID, accountSlug)
	} else {
		original, err = client.GetBitbucketAccount(installationID, orcaAccountID)
	}
	if err != nil {
		t.Fatalf("failed to snapshot bitbucket account: %s", err)
	}
	if original == nil {
		t.Skip("bitbucket account not found; cannot run adopt test")
	}
	// Destroy tears down the account's integrated repositories, and the restore
	// helper re-integrates only the empty unit. Require a disposable empty account
	// so the test never drops real repository integrations from a shared lab.
	if original.IntegratedRepositoriesCount > 0 {
		t.Skipf("bitbucket account %s has %d integrated repositories; point ORCA_TEST_BB_* at a disposable empty account (destroy removes repositories and they are not restored)",
			original.ID, original.IntegratedRepositoriesCount)
	}
	// Destroy DELETEs the Orca Bitbucket account; restore only re-integrates the empty unit.
	if os.Getenv("ORCA_TEST_BB_ALLOW_DESTROY") == "" {
		t.Skip("ORCA_TEST_BB_ALLOW_DESTROY not set; refuse to DELETE a shared lab Bitbucket account")
	}
	accountSlug = original.AccountID
	t.Cleanup(func() {
		restoreBitbucketAccount(t, client, installationID, accountSlug, original)
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_bitbucket_account" "t" {
  installation_id = %q
  account_id      = %q
  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}`, installationID, accountSlug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_bitbucket_account.t", "installation_id", installationID),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_bitbucket_account.t", "account_id", accountSlug),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_bitbucket_account.t", "configuration_settings.pr_summary_comment", "ONLY_ON_FAILED_ISSUES"),
				),
			},
			{
				ResourceName:      "orcasecurity_shift_left_bitbucket_account.t",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s/%s", installationID, accountSlug), nil
				},
			},
		},
	})
}

func restoreBitbucketAccount(t *testing.T, client *api_client.APIClient, installationID, accountSlug string, original *api_client.BitbucketAccount) {
	t.Helper()
	body := acctest.RestoreScmBody(original.ScmUnitCommonFields)
	cur, err := client.FindBitbucketAccountBySlug(installationID, accountSlug)
	if err != nil {
		t.Errorf("restore lookup: %s", err)
		return
	}
	if cur == nil {
		if err := client.IntegrateBitbucketUnit(api_client.BitbucketUnitIntegrate{
			InstallationID: installationID,
			AccountID:      accountSlug,
			Body:           body,
		}); err != nil {
			t.Errorf("failed to re-integrate bitbucket account %q: %s", accountSlug, err)
		}
		return
	}
	if _, err := client.UpdateBitbucketAccount(installationID, cur.ID, body); err != nil {
		if err2 := client.IntegrateBitbucketUnit(api_client.BitbucketUnitIntegrate{
			InstallationID: installationID,
			AccountID:      accountSlug,
			Body:           body,
		}); err2 != nil {
			t.Errorf("failed to restore bitbucket account %s (update: %v; re-integrate: %v)", cur.ID, err, err2)
		}
	}
}
