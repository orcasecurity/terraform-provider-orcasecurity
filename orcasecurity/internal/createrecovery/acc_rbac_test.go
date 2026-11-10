package createrecovery_test

import (
	"fmt"
	"strconv"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func inviteEmail(name string) string { return name + "@example.com" }

func fakeAWSAccountID(name string) string {
	n, _ := strconv.Atoi(name[strings.LastIndex(name, "-")+1:])
	return fmt.Sprintf("%012d", n)
}

func groupAccessIDs(c *api_client.APIClient, name string) (map[string]bool, error) {
	groups, err := c.ListGroupIDsByName(name)
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for gid := range groups {
		rows, err := c.ListGroupAccessForGroup(gid)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			ids[r.ID] = true
		}
	}
	return ids, nil
}

func init() {
	accCases = append(accCases,
		accCase{
			name: "business_unit", address: "orcasecurity_business_unit.t", path: "/api/business_units",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_business_unit" "t" {
  name        = %q
  filter_data = { cloud_providers = ["aws"] }
}`, n)
			},
			ids: (*api_client.APIClient).ListBusinessUnitIDsByName,
		},
		accCase{
			name: "custom_role", address: "orcasecurity_custom_role.t", path: "/api/rbac/roles",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_role" "t" {
  name              = %q
  description       = "WASP-1876 ambiguous-create acceptance test"
  permission_groups = ["assets.asset.read"]
}`, n)
			},
			ids: (*api_client.APIClient).ListCustomRoleIDsByName,
		},
		accCase{
			name: "group", address: "orcasecurity_group.t", path: "/api/rbac/group",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_group" "t" {
  name        = %q
  description = "WASP-1876 ambiguous-create acceptance test"
  sso_group   = false
}`, n)
			},
			ids: (*api_client.APIClient).ListGroupIDsByName,
		},
		accCase{
			name: "user_invite", address: "orcasecurity_add_users.t", path: "/api/user_invites/bulk_create/",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_role" "r" {
  name              = %q
  description       = "WASP-1876 ambiguous-create acceptance test"
  permission_groups = ["assets.asset.read"]
}

resource "orcasecurity_add_users" "t" {
  email             = %q
  role_id           = orcasecurity_custom_role.r.id
  should_send_email = false
}`, n, inviteEmail(n))
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.ListUserInviteIDsByEmail(inviteEmail(n))
			},
		},
		accCase{
			name: "trusted_cloud_account", address: "orcasecurity_trusted_cloud_account.t", path: "/api/organization/trusted_accounts",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_trusted_cloud_account" "t" {
  account_name      = %q
  description       = "WASP-1876 ambiguous-create acceptance test (fake account id)"
  cloud_provider    = "aws"
  cloud_provider_id = %q
}`, n, fakeAWSAccountID(n))
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.ListTrustedCloudAccountIDs("aws", fakeAWSAccountID(n))
			},
		},
		accCase{
			name: "group_access", address: "orcasecurity_group_access.t", path: "/api/rbac/access/group",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_role" "r" {
  name              = %q
  description       = "WASP-1876 ambiguous-create acceptance test"
  permission_groups = ["assets.asset.read"]
}

resource "orcasecurity_group" "g" {
  name        = %q
  description = "WASP-1876 ambiguous-create acceptance test"
  sso_group   = false
}

resource "orcasecurity_group_access" "t" {
  group_id           = orcasecurity_group.g.id
  role_id            = orcasecurity_custom_role.r.id
  all_cloud_accounts = true
}`, n, n)
			},
			ids: groupAccessIDs,
		},
	)
}
