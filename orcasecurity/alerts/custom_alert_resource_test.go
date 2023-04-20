package alerts_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomAlertResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_alert" "test" {
  name = "test name"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
  allow_adjusting = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "name", "test name"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "description", "test description"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "rule", "ActivityLogDetection"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "score", "5.5"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "category", "Best practices"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "allow_adjusting", "false"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_alert.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_alert.test", "organization_id"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
			resource "orcasecurity_custom_alert" "test" {
				name = "test name updated"
				description = "test description updated"
				rule = "Address"
				score = 9.5
				category = "Malware"
				allow_adjusting = true
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "name", "test name updated"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "description", "test description updated"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "rule", "Address"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "score", "9.5"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "category", "Malware"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "allow_adjusting", "true"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_alert.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_alert.test", "organization_id"),
				),
			},
		},
	})
}

func TestAccCustomAlertResource_AddRemediationText(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_alert" "test" {
  name = "test name2"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "remediation_text.enable", "false"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "remediation_text.text", ""),
				),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_alert" "test" {
					name = "test name2"
					description = "test description"
					rule = "ActivityLogDetection"
					score = 5.5
					category = "Best practices"
					remediation_text = {
						enable = true
						text   = "test text"
				   }
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "remediation_text.enable", "true"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "remediation_text.text", "test text"),
				),
			},
		},
	})
}

func TestAccCustomAlertResource_WithRemediationText(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_alert" "test_with_remediation" {
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
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test_with_remediation", "remediation_text.enable", "true"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test_with_remediation", "remediation_text.text", "test text"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_alert.test_with_remediation",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_alert" "test_with_remediation" {
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
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test_with_remediation", "remediation_text.enable", "false"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test_with_remediation", "remediation_text.text", "test text update"),
				),
			},
		},
	})
}

func TestAccCustomAlertResource_DeleteRemediationText(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_alert" "test_with_remediation" {
  name = "test name2"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
  remediation_text = {
	   enable = true
	   text   = "test text"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test_with_remediation", "remediation_text.enable", "true"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test_with_remediation", "remediation_text.text", "test text"),
				),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_alert" "test_with_remediation" {
					name = "test name2"
					description = "test description"
					rule = "ActivityLogDetection"
					score = 5.5
					category = "Best practices"
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test_with_remediation", "remediation_text.enable", "false"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test_with_remediation", "remediation_text.text", ""),
				),
			},
		},
	})
}

func TestAccCustomAlertResource_WithComplianceFramework(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_alert" "test" {
  name = "test name2"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
  compliance_frameworks = [
     { name = "test_terraform", section = "section_1", priority = "low" }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.name", "test_terraform"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.section", "section_1"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.priority", "low"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_alert" "test" {
					name = "test name2"
					description = "test description"
					rule = "ActivityLogDetection"
					score = 5.5
					category = "Best practices"
					compliance_frameworks = [
						{ name = "test_terraform", section = "section_2", priority = "medium" }
					 ]
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.name", "test_terraform"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.section", "section_2"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.priority", "medium"),
				),
			},
		},
	})
}

func TestAccCustomAlertResource_AddComplianceFramework(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_alert" "test" {
  name = "test name2"
  description = "test description"
  rule = "ActivityLogDetection"
  score = 5.5
  category = "Best practices"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.name", "test_terraform"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.section", "section_1"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.priority", "low"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_alert" "test" {
					name = "test name2"
					description = "test description"
					rule = "ActivityLogDetection"
					score = 5.5
					category = "Best practices"
					compliance_frameworks = [
						{ name = "test_terraform", section = "section_2", priority = "medium" }
					 ]
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.name", "test_terraform"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.section", "section_2"),
					resource.TestCheckResourceAttr("orcasecurity_custom_alert.test", "compliance_frameworks.0.priority", "medium"),
				),
			},
		},
	})
}
