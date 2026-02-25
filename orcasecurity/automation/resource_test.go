package automation_test

import (
	"regexp"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	testAccAutomationResourceName = "orcasecurity_automation.test"
	testAccAutomationName         = "test name"
	testAccAutomationNameUpdated  = "test name updated"
	testAccAutomationDesc         = "test description"
	testAccAutomationDescUpdated  = "test description updated"
)

func TestAccAutomationResource_RequireAtLeastOneIntegration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation" "test" {
  name        = "` + testAccAutomationName + `"
  description = "` + testAccAutomationDesc + `"
  enabled     = true
  query = {
    filter = [
      { field = "state.status", includes = ["open"] },
      { field = "state.risk_level", excludes = ["high"] }
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
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation" "test" {
  name        = "` + testAccAutomationName + `"
  description = "` + testAccAutomationDesc + `"
  enabled     = true
  query = {
    filter = [
      { field = "state.status", includes = ["open"] },
      { field = "state.risk_level", excludes = ["high"] }
    ]
  }
  jira_cloud_template = {
    template     = "tf: example"
    parent_issue = "FOO-1"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "name", testAccAutomationName),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "description", testAccAutomationDesc),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "jira_cloud_template.template", "tf: example"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "jira_cloud_template.parent_issue", "FOO-1"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "query.filter.0.field", "state.status"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "query.filter.0.includes.0", "open"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "query.filter.1.field", "state.risk_level"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "query.filter.1.excludes.0", "high"),
					resource.TestCheckResourceAttrSet(testAccAutomationResourceName, "id"),
					resource.TestCheckResourceAttrSet(testAccAutomationResourceName, "organization_id"),
				),
			},
			{
				ResourceName:      testAccAutomationResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation" "test" {
  name        = "` + testAccAutomationNameUpdated + `"
  description = "` + testAccAutomationDescUpdated + `"
  enabled     = true
  query = {
    filter = [
      { field = "state.status", includes = ["closed"] },
      { field = "state.risk_level", excludes = ["low"] },
      { field = "asset_regions", excludes = ["centralus"] }
    ]
  }
  jira_cloud_template = {
    template     = "tf: example updated"
    parent_issue = "FOO-2"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "name", testAccAutomationNameUpdated),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "description", testAccAutomationDescUpdated),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "jira_cloud_template.template", "tf: example updated"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "jira_cloud_template.parent_issue", "FOO-2"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "query.filter.0.field", "state.status"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "query.filter.0.includes.0", "closed"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "query.filter.1.field", "state.risk_level"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "query.filter.1.excludes.0", "low"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "query.filter.2.field", "asset_regions"),
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "query.filter.2.excludes.0", "centralus"),
					resource.TestCheckResourceAttrSet(testAccAutomationResourceName, "id"),
					resource.TestCheckResourceAttrSet(testAccAutomationResourceName, "organization_id"),
				),
			},
		},
	})
}

// Test resource with Sumo Logic integration
func TestAccAutomationResource_SumoLogic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation" "test" {
  name        = "` + testAccAutomationName + `"
  description = "` + testAccAutomationDesc + `"
  enabled     = true
  query = {
    filter = [
      { field = "state.status", includes = ["open"] }
    ]
  }
  sumo_logic_template = {}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(testAccAutomationResourceName, "id"),
				),
			},
			{
				ResourceName:      testAccAutomationResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation" "test" {
  name        = "` + testAccAutomationNameUpdated + `"
  description = "` + testAccAutomationDescUpdated + `"
  enabled     = true
  query = {
    filter = [
      { field = "state.status", includes = ["closed"] }
    ]
  }
  jira_cloud_template = {
    template     = "tf: example updated"
    parent_issue = "FOO-2"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "name", testAccAutomationNameUpdated),
				),
			},
		},
	})
}

// Test resource with web hook integration
func TestAccAutomationResource_Webhook(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation" "test" {
  name        = "` + testAccAutomationName + `"
  description = "` + testAccAutomationDesc + `"
  enabled     = true
  query = {
    filter = [
      { field = "state.status", includes = ["open"] }
    ]
  }
  webhook_template = {
    template = "tf_test"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "webhook_template.template", "tf_test"),
				),
			},
			{
				ResourceName:      testAccAutomationResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation" "test" {
  name        = "` + testAccAutomationNameUpdated + `"
  description = "` + testAccAutomationDescUpdated + `"
  enabled     = true
  query = {
    filter = [
      { field = "state.status", includes = ["closed"] }
    ]
  }
  sumo_logic_template = {}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(testAccAutomationResourceName, "name", testAccAutomationNameUpdated),
				),
			},
		},
	})
}
