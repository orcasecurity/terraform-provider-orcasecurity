package createrecovery_test

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func init() {
	accCases = append(accCases,
		accCase{
			name: "business_unit", address: "orcasecurity_business_unit.t", post: "/api/business_units",
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
			name: "custom_role", address: "orcasecurity_custom_role.t", post: "/api/rbac/roles",
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
			name: "group", address: "orcasecurity_group.t", post: "/api/rbac/group",
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
			name: "custom_tag_rule", address: "orcasecurity_custom_tag_rule.t", post: "/api/custom_tags",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_tag_rule" "t" {
  name    = %q
  rule     = "AwsEc2Instance with PublicIps"
  disabled = true
  tags     = { "wasp-1876" = "acc" }
}`, n)
			},
			ids: (*api_client.APIClient).ListCustomTagRuleIDsByName,
		},
		accCase{
			name: "discovery_view", address: "orcasecurity_discovery_view.t", post: "/api/user_preferences",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_discovery_view" "t" {
  name               = %q
  organization_level = false
  view_type          = "discovery"
  extra_params       = {}
  filter_data = {
    query = jsonencode({ "models" : ["AwsS3Bucket"], "type" : "object_set" })
  }
}`, n)
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.DiscoveryViewCreateOp(api_client.DiscoveryView{Name: n, ViewType: "discovery"}).IDs()
			},
		},
		accCase{
			name: "shift_left_cve_exception_list", address: "orcasecurity_shift_left_cve_exception_list.t", post: "/api/shiftleft/exceptions/",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_shift_left_cve_exception_list" "t" {
  name        = %q
  description = "WASP-1876 ambiguous-create acceptance test"
  disabled    = false
  vulnerabilities = [{
    cve_id      = "cve-2021-44228"
    description = "acc"
    disabled    = false
  }]
}`, n)
			},
			ids: (*api_client.APIClient).ListShiftLeftCveExceptionListIDsByName,
		},
	)
}
