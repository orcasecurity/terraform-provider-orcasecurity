package shift_left_installation_test

// Live reads of the three installation-list data sources. Creating an
// installation needs an SCM PAT the test environment does not hold, but
// listing them is GET-only, so these run against any tenant with credentials —
// zero rows is a pass (the check only requires the collection to exist).

import (
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccInstallationsDataSource(t *testing.T, typeName string) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "` + typeName + `" "all" {}
`,
				Check: resource.TestCheckResourceAttrSet("data."+typeName+".all", "installations.#"),
			},
		},
	})
}

func TestAccGitlabInstallationsDataSource_basic(t *testing.T) {
	testAccInstallationsDataSource(t, "orcasecurity_shift_left_gitlab_installations")
}

func TestAccBitbucketInstallationsDataSource_basic(t *testing.T) {
	testAccInstallationsDataSource(t, "orcasecurity_shift_left_bitbucket_installations")
}

func TestAccAzureDevopsInstallationsDataSource_basic(t *testing.T) {
	testAccInstallationsDataSource(t, "orcasecurity_shift_left_azure_devops_installations")
}
