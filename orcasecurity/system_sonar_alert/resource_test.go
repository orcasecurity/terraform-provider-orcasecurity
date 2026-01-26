package system_sonar_alert_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	ResourceType = "orcasecurity_system_sonar_alert"
	Resource     = "test"
	TestAlertID  = "r8ae477067a"
)

func TestAccSystemSonarAlertResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_system_sonar_alert" "test" {
  rule_id = "r8ae477067a"
  enabled = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_system_sonar_alert.test", "rule_id", "r8ae477067a"),
					resource.TestCheckResourceAttr("orcasecurity_system_sonar_alert.test", "enabled", "false"),
					resource.TestCheckResourceAttrSet("orcasecurity_system_sonar_alert.test", "name"),
					resource.TestCheckResourceAttrSet("orcasecurity_system_sonar_alert.test", "category"),
					resource.TestCheckResourceAttrSet("orcasecurity_system_sonar_alert.test", "score"),
					resource.TestCheckResourceAttrSet("orcasecurity_system_sonar_alert.test", "rule_type"),
				),
			},
			// Import
			{
				ResourceName:                         "orcasecurity_system_sonar_alert.test",
				ImportState:                          true,
				ImportStateId:                        "r8ae477067a",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "rule_id",
			},
			// Update (enable the alert)
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_system_sonar_alert" "test" {
  rule_id = "r8ae477067a"
  enabled = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_system_sonar_alert.test", "rule_id", "r8ae477067a"),
					resource.TestCheckResourceAttr("orcasecurity_system_sonar_alert.test", "enabled", "true"),
				),
			},
		},
	})
}

func TestAccSystemSonarAlertResource_EnableToggle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_system_sonar_alert" "test" {
  rule_id = "r8ae477067a"
  enabled = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_system_sonar_alert.test", "enabled", "true"),
				),
			},
			// Toggle to disabled
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_system_sonar_alert" "test" {
  rule_id = "r8ae477067a"
  enabled = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_system_sonar_alert.test", "enabled", "false"),
				),
			},
			// Toggle back to enabled
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_system_sonar_alert" "test" {
  rule_id = "r8ae477067a"
  enabled = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_system_sonar_alert.test", "enabled", "true"),
				),
			},
		},
	})
}
