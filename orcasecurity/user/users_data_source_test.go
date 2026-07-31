package user_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testAccDataSourceUsers = orcasecurity.TestProviderConfig + `
data "orcasecurity_users" "all" {}
`

func usersNestedIDCount(attr map[string]string) int {
	n := 0
	for k := range attr {
		if strings.HasPrefix(k, "users.") && strings.HasSuffix(k, ".user_id") {
			n++
		}
	}
	return n
}

func errIfInvalidUsersHash(n string) error {
	c, err := strconv.Atoi(n)
	if err != nil {
		return fmt.Errorf("users.#: %w", err)
	}
	if c < 1 {
		return fmt.Errorf("expected at least one user, got users.#=%d", c)
	}
	return nil
}

func errIfUsersAttrsEmpty(attr map[string]string) error {
	if n, ok := attr["users.#"]; ok {
		return errIfInvalidUsersHash(n)
	}
	if usersNestedIDCount(attr) < 1 {
		return fmt.Errorf("expected users in state; attributes: %#v", attr)
	}
	return nil
}

func testAccCheckUsersListNonEmpty(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource %q not found in state", name)
		}
		return errIfUsersAttrsEmpty(rs.Primary.Attributes)
	}
}

func TestAccUsersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceUsers,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckUsersListNonEmpty("data.orcasecurity_users.all"),
				),
			},
		},
	})
}
