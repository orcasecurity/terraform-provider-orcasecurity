package group_permissions_test

import (
	"regexp"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupPermissionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + testAccGroupPermissionResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_group_permission.test", "all_cloud_accounts", "true"),
					resource.TestCheckResourceAttr("orcasecurity_group_permission.test", "cloud_accounts.#", "0"),
					resource.TestCheckResourceAttr("orcasecurity_group_permission.test", "business_units.#", "0"),
					resource.TestCheckResourceAttr("orcasecurity_group_permission.test", "shiftleft_projects.#", "0"),
					resource.TestCheckResourceAttrSet("orcasecurity_group_permission.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "orcasecurity_group_permission.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing (expecting an error as API does not support update)
			{
				Config:      orcasecurity.TestProviderConfig + testAccGroupPermissionResourceUpdateConfig(),
				ExpectError: regexp.MustCompile("Update operation not supported by API"),
			},
			// Delete testing (implicitly covered by the test framework)
		},
	})
}

func testAccGroupPermissionResourceConfig() string {
	return `
resource "orcasecurity_group" "test" {
	name = "test-group-for-perms"
	description = "test-description"
	sso_group = false
	users = ["d8b1b3b4-8b1b-4b1b-8b1b-8b1b3b4b1b3b"]
}

resource "orcasecurity_custom_role" "test" {
	name = "test-role-for-perms"
	description = "test-description"
	permission_groups = ["assets.asset.read"]
}

resource "orcasecurity_group_permission" "test" {
	group_id = orcasecurity_group.test.id
	role_id = orcasecurity_custom_role.test.id
	all_cloud_accounts = true
}
`
}

func testAccGroupPermissionResourceUpdateConfig() string {
	return `
resource "orcasecurity_group" "test" {
	name = "test-group-for-perms"
	description = "test-description"
	sso_group = false
	users = ["d8b1b3b4-8b1b-4b1b-8b1b-8b1b3b4b1b3b"]
}

resource "orcasecurity_custom_role" "test" {
	name = "test-role-for-perms"
	description = "test-description"
	permission_groups = ["assets.asset.read"]
}

resource "orcasecurity_group_permission" "test" {
	group_id = orcasecurity_group.test.id
	role_id = orcasecurity_custom_role.test.id
	all_cloud_accounts = false
	cloud_accounts = ["d8b1b3b4-8b1b-4b1b-8b1b-8b1b3b4b1b3b"]
}
`
}
