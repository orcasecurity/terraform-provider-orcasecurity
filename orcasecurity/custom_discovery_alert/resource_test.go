package custom_discovery_alert_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

<<<<<<< HEAD
func TestAccCustomAlertResource_Basic(t *testing.T) {
=======
func TestAccCustomDiscoveryAlertResource_Basic(t *testing.T) {
>>>>>>> alert-docs-update
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "test name"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  orca_score = 5.5
<<<<<<< HEAD
  severity = 1
=======
>>>>>>> alert-docs-update
  category = "Best practices"
  context_score = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "name", "test name"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "description", "test description"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "orca_score", "5.5"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "category", "Best practices"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "context_score", "false"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_discovery_alert.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_discovery_alert.test", "organization_id"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_discovery_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
			resource "orcasecurity_custom_discovery_alert" "test" {
				name = "test name updated"
				description = "test description updated"
				rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
				orca_score = 9.5
<<<<<<< HEAD
				severity = 1
=======
>>>>>>> alert-docs-update
				context_score = true
				category = "Malware"
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "name", "test name updated"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "description", "test description updated"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "orca_score", "9.5"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "category", "Malware"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_discovery_alert.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_discovery_alert.test", "organization_id"),
				),
			},
		},
	})
}

<<<<<<< HEAD
func TestAccCustomAlertResource_AddRemediationText(t *testing.T) {
=======
func TestAccCustomDiscoveryAlertResource_AddRemediationText(t *testing.T) {
>>>>>>> alert-docs-update
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "test name2"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  context_score = true
  orca_score = 5.5
<<<<<<< HEAD
  severity = 1
=======
>>>>>>> alert-docs-update
  category = "Best practices"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("orcasecurity_custom_discovery_alert.test", "remediation_text"),
				),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_discovery_alert" "test" {
					name = "test name2"
					description = "test description"
					rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
					orca_score = 5.5
					category = "Best practices"
					context_score = true
					remediation_text = {
						enable = true
						text   = "test text"
				   }
				}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "remediation_text.enable", "true"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "remediation_text.text", "test text"),
				),
			},
		},
	})
}

<<<<<<< HEAD
func TestAccCustomAlertResource_UpdateRemediationText(t *testing.T) {
=======
func TestAccCustomDiscoveryAlertResource_UpdateRemediationText(t *testing.T) {
>>>>>>> alert-docs-update
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "test name2"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  orca_score = 5.5
<<<<<<< HEAD
  severity = 1
=======
>>>>>>> alert-docs-update
  category = "Best practices"
  context_score = true
  remediation_text = {
	   enable = true
	   text   = "test text"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "remediation_text.enable", "true"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "remediation_text.text", "test text"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_discovery_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_discovery_alert" "test" {
					name = "test name2"
					description = "test description"
					rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
					orca_score = 5.5
					category = "Best practices"
					context_score = true
					remediation_text = {
						 enable = false
						 text   = "test text update"
					}
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "remediation_text.enable", "false"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "remediation_text.text", "test text update"),
				),
			},
		},
	})
}

<<<<<<< HEAD
func TestAccCustomAlertResource_DeleteRemediationText(t *testing.T) {
=======
func TestAccCustomDiscoveryAlertResource_DeleteRemediationText(t *testing.T) {
>>>>>>> alert-docs-update
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "test name2"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  orca_score = 5.5
<<<<<<< HEAD
  severity = 1
=======
>>>>>>> alert-docs-update
  category = "Best practices"
  context_score = true
  remediation_text = {
	   enable = true
	   text   = "test text"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "remediation_text.enable", "true"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "remediation_text.text", "test text"),
				),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_discovery_alert" "test" {
					name = "test name2"
					description = "test description"
					rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
					orca_score = 5.5
<<<<<<< HEAD
					severity = 1
=======
			
>>>>>>> alert-docs-update
					category = "Best practices"
					context_score = true
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("orcasecurity_custom_discovery_alert.test", "remediation_text"),
				),
			},
		},
	})
}

<<<<<<< HEAD
func TestAccCustomAlertResource_AddComplianceFramework(t *testing.T) {
=======
func TestAccCustomDiscoveryAlertResource_AddComplianceFramework(t *testing.T) {
>>>>>>> alert-docs-update
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "test name2"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  orca_score = 5.5
<<<<<<< HEAD
  severity = 1
=======
>>>>>>> alert-docs-update
  category = "Best practices"
  context_score = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_discovery_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_discovery_alert" "test" {
					name = "test name2"
					description = "test description"
					rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
					orca_score = 5.5
<<<<<<< HEAD
					severity = 1
=======
			
>>>>>>> alert-docs-update
					category = "Best practices"
					context_score = true
					compliance_frameworks = [
						{ name = "test_terraform", section = "section_2", priority = "medium" }
					 ]
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.name", "test_terraform"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.section", "section_2"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.priority", "medium"),
				),
			},
		},
	})
}

<<<<<<< HEAD
func TestAccCustomAlertResource_UpdateComplianceFramework(t *testing.T) {
=======
func TestAccCustomDiscoveryAlertResource_UpdateComplianceFramework(t *testing.T) {
>>>>>>> alert-docs-update
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "test name2"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  orca_score = 5.5
<<<<<<< HEAD
  severity = 1
=======
>>>>>>> alert-docs-update
  category = "Best practices"
  context_score = true
  compliance_frameworks = [
	{ name = "test_terraform", section = "section_1", priority = "medium" }
 ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.name", "test_terraform"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.section", "section_1"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.priority", "medium"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_discovery_alert.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_discovery_alert" "test" {
					name = "test name2"
					description = "test description"
					rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
					orca_score = 5.5
<<<<<<< HEAD
					severity = 1
=======
			
>>>>>>> alert-docs-update
					category = "Best practices"
					context_score = true
					compliance_frameworks = [
						{ name = "test_terraform", section = "section_2", priority = "low" }
					 ]
				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.name", "test_terraform"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.section", "section_2"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.priority", "low"),
				),
			},
		},
	})
}

<<<<<<< HEAD
func TestAccCustomAlertResource_DeleteComplianceFramework(t *testing.T) {
=======
func TestAccCustomDiscoveryAlertResource_DeleteComplianceFramework(t *testing.T) {
>>>>>>> alert-docs-update
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "test name2"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  orca_score = 5.5
<<<<<<< HEAD
  severity = 1
=======
>>>>>>> alert-docs-update
  category = "Best practices"
  context_score = true
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
				resource "orcasecurity_custom_discovery_alert" "test" {
					name = "test name2"
					description = "test description"
					rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
					orca_score = 5.5
<<<<<<< HEAD
					severity = 1
=======
			
>>>>>>> alert-docs-update
					category = "Best practices"
					context_score = true

				  }
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks"),
				),
			},
		},
	})
}
