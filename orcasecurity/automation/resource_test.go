package automation_test

import (
	"regexp"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAutomationResource_RequireAtLeastOneIntegration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation" "test" {
  name = "test name"
  description = "test description"
  query = {
	filter: [
		{ field: "state.status", includes: ["open"] },
		{ field: "state.risk_level", excludes: ["high"] }
	]
  }
}
`,
				ExpectError: regexp.MustCompile("At least one of these attributes must be configured"),
			},
		},
	})
}

// Test resource with Jira issue settings and common attributes
// Note, API server must contain two Jira templates configured: "example" and "example updated"
func TestAccAutomationResource_JiraIssue(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation" "test" {
  name = "test name"
  description = "test description"
  query = {
	filter: [
		{ field: "state.status", includes: ["open"] },
		{ field: "state.risk_level", excludes: ["high"] }
	]
  }
  jira_issue = {
	template_name = "tf: example"
	parent_issue = "FOO-1"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify first order item
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "name", "test name"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "description", "test description"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "jira_issue.template_name", "tf: example"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "jira_issue.parent_issue", "FOO-1"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "query.filter.0.field", "state.status"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "query.filter.0.includes.0", "open"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "query.filter.1.field", "state.risk_level"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "query.filter.1.excludes.0", "high"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("orcasecurity_automation.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_automation.test", "organization_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "orcasecurity_automation.test",
				ImportState:       true,
				ImportStateVerify: true,
				// ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Update and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
			resource "orcasecurity_automation" "test" {
				name = "test name updated"
				description = "test description updated"
				query = {
				  filter: [
					  { field: "state.status", includes: ["closed"] },
					  { field: "state.risk_level", excludes: ["low"] },
					  { field: "asset_regions", excludes: ["centralus"] }
				  ]
				}
				jira_issue = {
				  template_name = "tf: example updated"
				  parent_issue = "FOO-2"
				}
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify first order item
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "name", "test name updated"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "description", "test description updated"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "jira_issue.template_name", "tf: example updated"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "jira_issue.parent_issue", "FOO-2"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "query.filter.0.field", "state.status"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "query.filter.0.includes.0", "closed"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "query.filter.1.field", "state.risk_level"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "query.filter.1.excludes.0", "low"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "query.filter.2.field", "asset_regions"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "query.filter.2.excludes.0", "centralus"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("orcasecurity_automation.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_automation.test", "organization_id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// Test resource with Sumo Logic integration
func TestAccAutomationResource_SumoLogic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation" "test" {
  name = "test name"
  description = "test description"
  query = {
	filter: [
		{ field: "state.status", includes: ["open"] },
	]
  }
  sumologic = {}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
				// sumologic has no attributes
				),
			},
			// ImportState testing
			{
				ResourceName:      "orcasecurity_automation.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing (deactivate sumologic)
			{
				Config: orcasecurity.TestProviderConfig + `
			resource "orcasecurity_automation" "test" {
				name = "test name updated"
				description = "test description updated"
				query = {
				  filter: [
					  { field: "state.status", includes: ["closed"] },
				  ]
				}
				jira_issue = {
					template_name = "tf: example updated"
					parent_issue = "FOO-2"
				  }
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
				// sumologic has no attributes
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// Test resource with web hook integration
func TestAccAutomationResource_Webhook(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation" "test" {
  name = "test name"
  description = "test description"
  query = {
	filter: [
		{ field: "state.status", includes: ["open"] },
	]
  }
  webhook = {
	name = "tf_test"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
				// sumologic has no attributes
				),
			},
			// ImportState testing
			{
				ResourceName:      "orcasecurity_automation.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing (deactivate sumologic)
			{
				Config: orcasecurity.TestProviderConfig + `
			resource "orcasecurity_automation" "test" {
				name = "test name updated"
				description = "test description updated"
				query = {
				  filter: [
					  { field: "state.status", includes: ["closed"] },
				  ]
				}
				sumologic = {}
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "name", "test name updated"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
