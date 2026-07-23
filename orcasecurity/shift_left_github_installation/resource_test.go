package shift_left_github_installation_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubInstallation_import(t *testing.T) {
	id := os.Getenv("ORCA_TEST_GH_INSTALLATION_ID")
	if id == "" {
		t.Skip("ORCA_TEST_GH_INSTALLATION_ID not set")
	}
	// Destroy DELETEs the Orca GitHub installation (cannot re-create without App flow).
	if os.Getenv("ORCA_TEST_GH_ALLOW_DESTROY") == "" {
		t.Skip("ORCA_TEST_GH_ALLOW_DESTROY not set; refuse to DELETE a shared lab GitHub installation")
	}

	orcasecurity.TestAccPreCheck(t)
	client := acctest.APIClient(t)
	original, err := client.GetGithubInstallation(id)
	if err != nil {
		t.Fatalf("failed to snapshot github installation %s: %s", id, err)
	}
	if original == nil {
		t.Skipf("github installation %s not found; cannot run adopt test", id)
	}
	t.Cleanup(func() {
		cur, err := client.GetGithubInstallation(id)
		if err != nil || cur == nil {
			t.Logf("github installation %s deleted (expected); reinstall via GitHub App to restore", id)
			return
		}
		if _, err := client.UpdateGithubInstallation(id, acctest.RestoreScmBody(original.InstallationMode, original.DefaultPolicies, original.Policies, original.Project, original.ConfigSettings)); err != nil {
			t.Errorf("failed to restore github installation %s: %s", id, err)
		}
	})

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
