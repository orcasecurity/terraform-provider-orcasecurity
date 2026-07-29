package custom_discovery_alert_test

import (
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// frameworkName is scoped to this package because framework names are unique per org and the
// acceptance packages run concurrently.
const frameworkName = "tf-acc-discovery-framework"

// frameworkConfig provisions the custom framework that the compliance_frameworks tests attach the
// alert to. The alert API resolves frameworks by name and section, so the framework and both
// sections have to exist before an alert can reference them.
var frameworkConfig = fmt.Sprintf(`
resource "orcasecurity_custom_compliance_framework" "test_framework" {
    name        = %q
    description = "Framework fixture for custom discovery alert acceptance tests"
    sections = [
        {
            name  = "section_1"
            tests = [{ rule_id = "rc7bcf3b77f", rule_id_in_framework = "1" }]
        },
        {
            name  = "section_2"
            tests = [{ rule_id = "rc7bcf3b77f", rule_id_in_framework = "2" }]
        }
    ]
}
`, frameworkName)

func TestAccCustomDiscoveryAlertResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "disco test name"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  orca_score = 5.5
  category = "Best practices"
  context_score = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "name", "disco test name"),
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
				name = "disco test name updated"
				description = "test description updated"
				rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
				orca_score = 9.5
				context_score = true
				category = "Malware"
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "name", "disco test name updated"),
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

func TestAccCustomDiscoveryAlertResource_AddRemediationText(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "disco test name2"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  context_score = true
  orca_score = 5.5
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
					name = "disco test name2"
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

func TestAccCustomDiscoveryAlertResource_UpdateRemediationText(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "disco test name2"
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
					name = "disco test name2"
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

func TestAccCustomDiscoveryAlertResource_DeleteRemediationText(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "disco test name2"
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
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
				resource "orcasecurity_custom_discovery_alert" "test" {
					name = "disco test name2"
					description = "test description"
					rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
					orca_score = 5.5
			
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

func TestAccCustomDiscoveryAlertResource_AddComplianceFramework(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + frameworkConfig + `
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "disco test name2"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  orca_score = 5.5
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
				Config: orcasecurity.TestProviderConfig + frameworkConfig + fmt.Sprintf(`
				resource "orcasecurity_custom_discovery_alert" "test" {
					name = "disco test name2"
					description = "test description"
					rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
					orca_score = 5.5
			
					category = "Best practices"
					context_score = true
					compliance_frameworks = [
						{ name = %q, section = "section_2", priority = "medium" }
					 ]
					depends_on = [orcasecurity_custom_compliance_framework.test_framework]
				  }
			`, frameworkName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.name", frameworkName),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.section", "section_2"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.priority", "medium"),
				),
			},
		},
	})
}

func TestAccCustomDiscoveryAlertResource_UpdateComplianceFramework(t *testing.T) {
	alertConfig := func(section, priority string) string {
		return orcasecurity.TestProviderConfig + frameworkConfig + fmt.Sprintf(`
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "disco test name2"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  orca_score = 5.5
  category = "Best practices"
  context_score = true
  compliance_frameworks = [
	{ name = %q, section = %q, priority = %q }
 ]
  depends_on = [orcasecurity_custom_compliance_framework.test_framework]
}
`, frameworkName, section, priority)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: alertConfig("section_1", "medium"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.name", frameworkName),
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
				Config: alertConfig("section_2", "low"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.name", frameworkName),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.section", "section_2"),
					resource.TestCheckResourceAttr("orcasecurity_custom_discovery_alert.test", "compliance_frameworks.0.priority", "low"),
				),
			},
		},
	})
}

func TestAccCustomDiscoveryAlertResource_DeleteComplianceFramework(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + frameworkConfig + fmt.Sprintf(`
resource "orcasecurity_custom_discovery_alert" "test" {
  name = "disco test name2"
  description = "test description"
  rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
  orca_score = 5.5
  category = "Best practices"
  context_score = true
  compliance_frameworks = [
	{ name = %q, section = "section_2", priority = "medium" }
 ]
  depends_on = [orcasecurity_custom_compliance_framework.test_framework]
}
`, frameworkName),
				Check: resource.ComposeAggregateTestCheckFunc(),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + frameworkConfig + `
				resource "orcasecurity_custom_discovery_alert" "test" {
					name = "disco test name2"
					description = "test description"
					rule_json = jsonencode({"models":["AzureAksCluster"],"type":"object_set"})
					orca_score = 5.5
			
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
