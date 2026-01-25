package custom_sonar_alert_test

import (
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	ResourceType = "orcasecurity_custom_sonar_alert"
	Resource     = "terraformTestResource"
	OrcaObject   = "terraformTestResourceInOrca"
)

func TestAccCustomSonarAlertResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
  name = "%s"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
  allow_adjusting = false
}
`, ResourceType, Resource, OrcaObject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "name", OrcaObject),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "description", "test description"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "rule", "ActivityLogDetection"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "score", "5.5"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "category", "Best practices"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "allow_adjusting", "false"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_sonar_alert.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_sonar_alert.test", "organization_id"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_sonar_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
			resource "orcasecurity_custom_sonar_alert" "test" {
				name = "test name updated"
				description = "test description updated"
				rule = "Address"
				score = 9.5
				category = "Malware"
				allow_adjusting = true
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "name", "test name updated"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "description", "test description updated"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "rule", "Address"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "score", "9.5"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "category", "Malware"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "allow_adjusting", "true"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_sonar_alert.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_sonar_alert.test", "organization_id"),
				),
			},
		},
	})
}

func TestAccCustomSonarAlertResource_AddRemediationText(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name = "test name2"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
  allow_adjusting = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("orcasecurity_custom_sonar_alert.test", "remediation_text"),
				),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_sonar_alert" "test" {
					name = "test name2"
					description = "test description"
					rule = "ActivityLogDetection"
					score = 5.5
					category = "Best practices"
					allow_adjusting = true
					remediation_text = {
						enable = true
						text   = "test text"
				   }
				}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "remediation_text.enable", "true"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "remediation_text.text", "test text"),
				),
			},
		},
	})
}

func TestAccCustomSonarAlertResource_UpdateRemediationText(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name = "test name2"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
  allow_adjusting = false
  remediation_text = {
	   enable = true
	   text   = "test text"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "remediation_text.enable", "true"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "remediation_text.text", "test text"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_sonar_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_sonar_alert" "test" {
					name = "test name2"
					description = "test description"
					rule = "ActivityLogDetection"
					score = 5.5
					category = "Best practices"
					allow_adjusting = false
					remediation_text = {
						 enable = false
						 text   = "test text update"
					}
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "remediation_text.enable", "false"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "remediation_text.text", "test text update"),
				),
			},
		},
	})
}

func TestAccCustomSonarAlertResource_DeleteRemediationText(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name = "test name2"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
  allow_adjusting = false
  remediation_text = {
	   enable = true
	   text   = "test text"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "remediation_text.enable", "true"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "remediation_text.text", "test text"),
				),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_sonar_alert" "test" {
					name = "test name2"
					description = "test description"
					rule = "ActivityLogDetection"
					score = 5.5
					category = "Best practices"
					allow_adjusting = false
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("orcasecurity_custom_sonar_alert.test", "remediation_text"),
				),
			},
		},
	})
}

func TestAccCustomSonarAlertResource_AddComplianceFramework(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name = "test name2"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
  allow_adjusting = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("orcasecurity_custom_sonar_alert.test", "compliance_frameworks"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_sonar_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_sonar_alert" "test" {
					name = "test name2"
					description = "test description"
					rule = "ActivityLogDetection"
					score = 5.5
					category = "Best practices"
					allow_adjusting = true
					compliance_frameworks = [
						{ name = "test_terraform", section = "section_2", priority = "medium" }
					 ]
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "compliance_frameworks.0.name", "test_terraform"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "compliance_frameworks.0.section", "section_2"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "compliance_frameworks.0.priority", "medium"),
				),
			},
		},
	})
}

func TestAccCustomSonarAlertResource_UpdateComplianceFramework(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name = "test name2"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
  allow_adjusting = true
  compliance_frameworks = [
	{ name = "test_terraform", section = "section_1", priority = "medium" }
 ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "compliance_frameworks.0.name", "test_terraform"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "compliance_frameworks.0.section", "section_1"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "compliance_frameworks.0.priority", "medium"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_sonar_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_sonar_alert" "test" {
					name = "test name2"
					description = "test description"
					rule = "ActivityLogDetection"
					score = 5.5
					category = "Best practices"
					allow_adjusting = true
					compliance_frameworks = [
						{ name = "test_terraform", section = "section_2", priority = "low" }
					 ]
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "compliance_frameworks.0.name", "test_terraform"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "compliance_frameworks.0.section", "section_2"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "compliance_frameworks.0.priority", "low"),
				),
			},
		},
	})
}

func TestAccCustomSonarAlertResource_DeleteComplianceFramework(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name = "test name2"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
  allow_adjusting = true
  compliance_frameworks = [
	{ name = "test_terraform", section = "section_2", priority = "medium" }
 ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_sonar_alert" "test" {
					name = "test name2"
					description = "test description"
					rule = "ActivityLogDetection"
					score = 5.5
					category = "Best practices"
					allow_adjusting = true

				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("orcasecurity_custom_sonar_alert.test", "compliance_frameworks"),
				),
			},
		},
	})
}

func TestAccCustomSonarAlertResource_EnabledToggle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create with enabled = true (default)
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name          = "test enabled toggle"
  description   = "test description"
  rule          = "ActivityLogDetection"
  orca_score    = 5.5
  category      = "Best practices"
  context_score = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "name", "test enabled toggle"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "enabled", "true"),
				),
			},
			// update to disable
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name          = "test enabled toggle"
  description   = "test description"
  rule          = "ActivityLogDetection"
  orca_score    = 5.5
  category      = "Best practices"
  context_score = false
  enabled       = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "enabled", "false"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_sonar_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update to re-enable
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name          = "test enabled toggle"
  description   = "test description"
  rule          = "ActivityLogDetection"
  orca_score    = 5.5
  category      = "Best practices"
  context_score = false
  enabled       = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "enabled", "true"),
				),
			},
		},
	})
}

func TestAccCustomSonarAlertResource_CreateDisabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create with enabled = false
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name          = "test create disabled"
  description   = "test description"
  rule          = "ActivityLogDetection"
  orca_score    = 5.5
  category      = "Best practices"
  context_score = false
  enabled       = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "name", "test create disabled"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "enabled", "false"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_sonar_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccCustomSonarAlertResource_EnabledPreservedOnUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create with enabled = true (explicitly set)
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name          = "test enabled preserved"
  description   = "original description"
  rule          = "ActivityLogDetection"
  orca_score    = 5.5
  category      = "Best practices"
  context_score = false
  enabled       = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "name", "test enabled preserved"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "description", "original description"),
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "enabled", "true"),
				),
			},
			// update description WITHOUT specifying enabled - enabled should remain true
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name          = "test enabled preserved"
  description   = "updated description"
  rule          = "ActivityLogDetection"
  orca_score    = 5.5
  category      = "Best practices"
  context_score = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "description", "updated description"),
					// This is the critical check - enabled should still be true even though we didn't specify it
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "enabled", "true"),
				),
			},
			// update name again WITHOUT specifying enabled - enabled should still remain true
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_sonar_alert" "test" {
  name          = "test enabled preserved updated"
  description   = "updated description"
  rule          = "ActivityLogDetection"
  orca_score    = 5.5
  category      = "Best practices"
  context_score = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "name", "test enabled preserved updated"),
					// enabled should still be true
					resource.TestCheckResourceAttr("orcasecurity_custom_sonar_alert.test", "enabled", "true"),
				),
			},
		},
	})
}
