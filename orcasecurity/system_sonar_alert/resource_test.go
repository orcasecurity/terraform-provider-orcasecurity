package system_sonar_alert_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSystemSonarAlertResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_system_sonar_alert" "example" {
  id = "r86ccacce6e"
  enabled = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_system_sonar_alert.example", "id", "r86ccacce6e"),
					resource.TestCheckResourceAttr("orcasecurity_system_sonar_alert.example", "enabled", "false"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_system_sonar_alert.example",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_system_sonar_alert" "example" {
  id = "r86ccacce6e"
  enabled = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_system_sonar_alert.example", "id", "r86ccacce6e"),
					resource.TestCheckResourceAttr("orcasecurity_system_sonar_alert.example", "enabled", "true"),
				),
			},
		},
	})
}
