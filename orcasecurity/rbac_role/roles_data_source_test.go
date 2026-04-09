package rbac_role_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testAccDataSourceRbacRoles = orcasecurity.TestProviderConfig + `
data "orcasecurity_rbac_roles" "all" {}
`

func testAccCheckRbacRolesListNonEmpty(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource %q not found in state", name)
		}
		attr := rs.Primary.Attributes
		if n, ok := attr["roles.#"]; ok {
			c, err := strconv.Atoi(n)
			if err != nil {
				return fmt.Errorf("roles.#: %w", err)
			}
			if c < 1 {
				return fmt.Errorf("expected at least one role, got roles.#=%d", c)
			}
			return nil
		}
		// Fallback: count nested role ids if flattening differs
		count := 0
		for k := range attr {
			if strings.HasPrefix(k, "roles.") && strings.HasSuffix(k, ".id") {
				count++
			}
		}
		if count < 1 {
			return fmt.Errorf("expected roles in state; attributes: %#v", attr)
		}
		return nil
	}
}

func TestAccRbacRolesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRbacRoles,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRbacRolesListNonEmpty("data.orcasecurity_rbac_roles.all"),
				),
			},
		},
	})
}
