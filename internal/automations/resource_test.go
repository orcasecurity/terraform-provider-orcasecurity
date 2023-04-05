package automations_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAutomationJiraIssueResource(t *testing.T) {
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
	template_name = "example"
	parent_issue = "FOO-1"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify first order item
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "name", "test name"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "description", "test description"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "jira_issue.template_name", "example"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "jira_issue.parent_issue", "FOO-1"),
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
					  { field: "state.risk_level", excludes: ["low"] }
				  ]
				}
				jira_issue = {
				  template_name = "example updated"
				  parent_issue = "FOO-2"
				}
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify first order item
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "name", "test name updated"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "description", "test description updated"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "jira_issue.template_name", "example updated"),
					resource.TestCheckResourceAttr("orcasecurity_automation.test", "jira_issue.parent_issue", "FOO-2"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("orcasecurity_automation.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_automation.test", "organization_id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
