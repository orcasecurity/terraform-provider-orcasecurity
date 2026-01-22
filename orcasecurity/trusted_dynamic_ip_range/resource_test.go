package trusted_dynamic_ip_range_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDynamicTrustedIpRangeResource_EnabledToggle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with enabled = true
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_organization" "current" {}

resource "orcasecurity_dynamic_trusted_ip_range" "test" {
  org_id  = data.orcasecurity_organization.current.id
  enabled = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_dynamic_trusted_ip_range.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("orcasecurity_dynamic_trusted_ip_range.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_dynamic_trusted_ip_range.test", "org_id"),
				),
			},
			// Step 2: Update to enabled = false
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_organization" "current" {}

resource "orcasecurity_dynamic_trusted_ip_range" "test" {
  org_id  = data.orcasecurity_organization.current.id
  enabled = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_dynamic_trusted_ip_range.test", "enabled", "false"),
				),
			},
			// Step 3: Update back to enabled = true
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_organization" "current" {}

resource "orcasecurity_dynamic_trusted_ip_range" "test" {
  org_id  = data.orcasecurity_organization.current.id
  enabled = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_dynamic_trusted_ip_range.test", "enabled", "true"),
				),
			},
		},
	})
}

// TestAccDynamicTrustedIpRangeResource_CreateDisabled tests creating the resource
// with enabled = false from the start.
func TestAccDynamicTrustedIpRangeResource_CreateDisabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with enabled = false
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_organization" "current" {}

resource "orcasecurity_dynamic_trusted_ip_range" "test_disabled" {
  org_id  = data.orcasecurity_organization.current.id
  enabled = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_dynamic_trusted_ip_range.test_disabled", "enabled", "false"),
					resource.TestCheckResourceAttrSet("orcasecurity_dynamic_trusted_ip_range.test_disabled", "id"),
				),
			},
		},
	})
}
