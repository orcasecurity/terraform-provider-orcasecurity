package shift_left_github_account_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubAccount_import(t *testing.T) {
	id := os.Getenv("ORCA_TEST_GH_ACCOUNT_ID")
	if id == "" {
		t.Skip("ORCA_TEST_GH_ACCOUNT_ID not set")
	}
	// Destroy DELETEs the Orca GitHub account (cannot re-create without App flow).
	if os.Getenv("ORCA_TEST_GH_ALLOW_DESTROY") == "" {
		t.Skip("ORCA_TEST_GH_ALLOW_DESTROY not set; refuse to DELETE a shared lab GitHub account")
	}

	client := acctest.APIClient(t)
	original, err := client.GetGithubInstallation(id)
	if err != nil {
		t.Fatalf("failed to snapshot github account %s: %s", id, err)
	}
	if original == nil {
		t.Skipf("github account %s not found; cannot run adopt test", id)
	}
	// Destroy DELETEs the GitHub account unit and every integrated repository
	// under it. There is no Integrate restore path (reinstall needs the GitHub
	// App flow), so refuse to run against a unit that still has repositories.
	if original.IntegratedRepositoriesCount > 0 {
		t.Skipf("github account %s has %d integrated repositories; point ORCA_TEST_GH_ACCOUNT_ID at a disposable empty account (destroy removes repositories and they are not restored)",
			id, original.IntegratedRepositoriesCount)
	}
	t.Cleanup(func() {
		cur, err := client.GetGithubInstallation(id)
		if err != nil || cur == nil {
			t.Logf("github account %s deleted (expected); reinstall via GitHub App to restore", id)
			return
		}
		if _, err := client.UpdateGithubInstallation(id, acctest.RestoreScmBody(original.InstallationMode, original.DefaultPolicies, original.Policies, original.Project, original.ConfigSettings)); err != nil {
			t.Errorf("failed to restore github account %s: %s", id, err)
		}
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_github_account" "t" {
  account_id = %q
  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}`, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_github_account.t", "account_id", id),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_github_account.t", "configuration_settings.pr_summary_comment", "ONLY_ON_FAILED_ISSUES"),
				),
			},
			{
				ResourceName:      "orcasecurity_shift_left_github_account.t",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
