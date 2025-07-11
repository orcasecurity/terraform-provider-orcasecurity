package group_permissions_test

import (
	"fmt"
	"regexp"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupPermissionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + testAccGroupPermissionResourceConfig("test-group-id", "test-role-id", true, `[]`, `[]`, `[]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_group_permission.test", "group_id", "test-group-id"),
					resource.TestCheckResourceAttr("orcasecurity_group_permission.test", "role_id", "test-role-id"),
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
				Config: orcasecurity.TestProviderConfig + testAccGroupPermissionResourceConfig("test-group-id", "test-role-id", false, `["acc-123"]`, `["business-unit-1"]`, `["sl-project-1"]`),
				ExpectError: regexp.MustCompile("Update operation not supported by API"),
			},
			// Delete testing (implicitly covered by the test framework)
		},
	})
}

func testAccGroupPermissionResourceConfig(groupID, roleID string, allCloudAccounts bool, cloudAccounts, businessUnits, shiftleftProjects string) string {
	return fmt.Sprintf(`
resource "orcasecurity_group_permission" "test" {
	group_id = %q
	role_id = %q
	all_cloud_accounts = %t
	cloud_accounts = %s
	business_units = %s
	shiftleft_projects = %s
}
`, groupID, roleID, allCloudAccounts, cloudAccounts, businessUnits, shiftleftProjects)
}
