package createrecovery_test

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func init() {
	accCases = append(accCases,
		accCase{
			name: "custom_dashboard", address: "orcasecurity_custom_dashboard.t", path: "/api/user_preferences",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_dashboard" "t" {
  name               = %q
  filter_data        = {}
  organization_level = false
  view_type          = "dashboard"
  extra_params = {
    description    = "WASP-1876 ambiguous-create acceptance test"
    version        = 2
    widgets_config = [{ id = "cloud-accounts-inventory", size = "sm" }]
  }
}`, n)
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.CustomDashboardCreateOp(api_client.CustomDashboard{Name: n, ViewType: "dashboard"}).IDs()
			},
		},
		accCase{
			name: "custom_widget", address: "orcasecurity_custom_widget.t", path: "/api/user_preferences",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_widget" "t" {
  name               = %q
  organization_level = false
  extra_params = {
    type                = "donut"
    empty_state_message = "No data found"
    default_size        = "sm"
    is_new              = true
    subtitle            = "WASP-1876"
    description         = "WASP-1876 ambiguous-create acceptance test"
    settings = {
      field = {
        name = "Type"
        type = "str"
      }
      request_params = {
        query    = jsonencode({ models = ["AwsS3Bucket"], type = "object_set" })
        group_by = ["Type"]
      }
    }
  }
}`, n)
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.CustomWidgetCreateOp(api_client.CustomWidget{Name: n, ViewType: api_client.CustomWidgetViewType}).IDs()
			},
		},
		accCase{
			name: "discovery_view", address: "orcasecurity_discovery_view.t", path: "/api/user_preferences",
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
			name: "shift_left_project", address: "orcasecurity_shift_left_project.t", path: "/api/shiftleft/projects/",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_shift_left_project" "t" {
  name             = %[1]q
  description      = "WASP-1876 ambiguous-create acceptance test"
  key              = %[1]q
  default_policies = true
}`, n)
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.ListShiftLeftProjectIDsByKey(n, n)
			},
		},
		accCase{
			name: "shift_left_cve_exception_list", address: "orcasecurity_shift_left_cve_exception_list.t", path: "/api/shiftleft/exceptions/",
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
