package shift_left_repository_test

// Live reads of the four repository-list data sources. Repository
// create/update/delete stays stub-only (import_apply_test.go) because
// destroying an integrated repository deletes its repository context in the
// tenant, but listing is GET-only, so these run against any tenant with
// credentials — zero rows is a pass (the check only requires the collection
// to exist).

import (
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccRepositoriesDataSource(t *testing.T, typeName string) {
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
				Check: resource.TestCheckResourceAttrSet("data."+typeName+".all", "repositories.#"),
			},
		},
	})
}

func TestAccGithubRepositoriesDataSource_basic(t *testing.T) {
	testAccRepositoriesDataSource(t, "orcasecurity_shift_left_github_repositories")
}

func TestAccGitlabRepositoriesDataSource_basic(t *testing.T) {
	testAccRepositoriesDataSource(t, "orcasecurity_shift_left_gitlab_repositories")
}

func TestAccBitbucketRepositoriesDataSource_basic(t *testing.T) {
	testAccRepositoriesDataSource(t, "orcasecurity_shift_left_bitbucket_repositories")
}

func TestAccAzureDevopsRepositoriesDataSource_basic(t *testing.T) {
	testAccRepositoriesDataSource(t, "orcasecurity_shift_left_azure_devops_repositories")
}
