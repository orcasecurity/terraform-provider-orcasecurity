package automation_v2_test

import (
	"regexp"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAutomationV2Resource_RequireAtLeastOneAction(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test that at least one action is required
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test name v2"
  description = "test description v2"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
}
`,
				ExpectError: regexp.MustCompile("At least one of these attributes must be configured"),
			},
		},
	})
}

func TestAccAutomationV2Resource_RequireFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test that filter is required
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test name v2"
  description = "test description v2"
  status = "enabled"
  sumo_logic_template = {
    external_config_id = "test-uuid"
  }
}
`,
				ExpectError: regexp.MustCompile("filter is required"),
			},
		},
	})
}

// Test resource with Jira Cloud integration using external_config_id
func TestAccAutomationV2Resource_JiraCloud(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test name v2"
  description = "test description v2"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  jira_cloud_template = {
    external_config_id = "test-jira-config-uuid"
    parent_issue = "FOO-1"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "name", "test name v2"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "description", "test description v2"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "status", "enabled"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "jira_cloud_template.external_config_id", "test-jira-config-uuid"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "jira_cloud_template.parent_issue", "FOO-1"),
					// Verify dynamic values have any value set in the state
					resource.TestCheckResourceAttrSet("orcasecurity_automation_v2.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_automation_v2.test", "organization_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "orcasecurity_automation_v2.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test name v2 updated"
  description = "test description v2 updated"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  jira_cloud_template = {
    external_config_id = "test-jira-config-uuid-updated"
    parent_issue = "FOO-2"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "name", "test name v2 updated"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "description", "test description v2 updated"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "status", "enabled"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "jira_cloud_template.external_config_id", "test-jira-config-uuid-updated"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "jira_cloud_template.parent_issue", "FOO-2"),
					resource.TestCheckResourceAttrSet("orcasecurity_automation_v2.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_automation_v2.test", "organization_id"),
				),
			},
		},
	})
}

// Test resource with Sumo Logic integration using external_config_id
func TestAccAutomationV2Resource_SumoLogic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test sumo logic automation"
  description = "test sumo logic description"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  sumo_logic_template = {
    external_config_id = "test-sumo-config-uuid"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "name", "test sumo logic automation"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "sumo_logic_template.external_config_id", "test-sumo-config-uuid"),
				),
			},
		},
	})
}

// Test resource with Azure Sentinel integration using external_config_id
func TestAccAutomationV2Resource_AzureSentinel(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test azure sentinel automation"
  description = "test azure sentinel description"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  azure_sentinel_template = {
    external_config_id = "test-azure-sentinel-uuid"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "name", "test azure sentinel automation"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "azure_sentinel_template.external_config_id", "test-azure-sentinel-uuid"),
				),
			},
		},
	})
}

// Test resource with Slack integration using external_config_id
func TestAccAutomationV2Resource_Slack(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test slack automation"
  description = "test slack description"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  slack_template = {
    external_config_id = "test-slack-config-uuid"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "name", "test slack automation"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "slack_template.external_config_id", "test-slack-config-uuid"),
				),
			},
		},
	})
}

// Test resource with Datadog integration including type field
func TestAccAutomationV2Resource_Datadog(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with LOGS type
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test datadog automation"
  description = "test datadog description"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  datadog_template = {
    external_config_id = "test-datadog-config-uuid"
    type = "LOGS"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "name", "test datadog automation"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "datadog_template.external_config_id", "test-datadog-config-uuid"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "datadog_template.type", "LOGS"),
				),
			},
			// Update to EVENT type
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test datadog automation updated"
  description = "test datadog description updated"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  datadog_template = {
    external_config_id = "test-datadog-config-uuid-updated"
    type = "EVENT"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "name", "test datadog automation updated"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "datadog_template.external_config_id", "test-datadog-config-uuid-updated"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "datadog_template.type", "EVENT"),
				),
			},
		},
	})
}

// Test resource with invalid Datadog type
func TestAccAutomationV2Resource_DatadogInvalidType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test datadog automation"
  description = "test datadog description"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  datadog_template = {
    external_config_id = "test-datadog-config-uuid"
    type = "INVALID"
  }
}
`,
				ExpectError: regexp.MustCompile("value must be one of.*LOGS.*EVENT"),
			},
		},
	})
}

// Test resource with webhook integration using external_config_id
func TestAccAutomationV2Resource_Webhook(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test webhook automation"
  description = "test webhook description"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  webhook_template = {
    external_config_id = "test-webhook-config-uuid"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "name", "test webhook automation"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "webhook_template.external_config_id", "test-webhook-config-uuid"),
				),
			},
		},
	})
}

// Test resource with email template
func TestAccAutomationV2Resource_Email(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test email automation"
  description = "test email description"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  email_template = {
    email = ["test@example.com", "admin@example.com"]
    multi_alerts = true
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "name", "test email automation"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "email_template.email.0", "test@example.com"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "email_template.email.1", "admin@example.com"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "email_template.multi_alerts", "true"),
				),
			},
		},
	})
}

// Test resource with snooze template
func TestAccAutomationV2Resource_Snooze(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test snooze automation"
  description = "test snooze description"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  snooze_template = {
    days = 7
    reason = "Under investigation"
    justification = "Security team is currently investigating this issue"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "name", "test snooze automation"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "snooze_template.days", "7"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "snooze_template.reason", "Under investigation"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "snooze_template.justification", "Security team is currently investigating this issue"),
				),
			},
		},
	})
}

// Test resource with invalid JSON in sonar_query
func TestAccAutomationV2Resource_InvalidJSON(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test invalid json"
  description = "test invalid json"
  status = "enabled"
  filter = {
    sonar_query = "invalid json string"
  }
  sumo_logic_template = {
    external_config_id = "test-uuid"
  }
}
`,
				ExpectError: regexp.MustCompile("invalid sonar_query JSON"),
			},
		},
	})
}

// Test resource with end_time
func TestAccAutomationV2Resource_WithEndTime(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_automation_v2" "test" {
  name = "test automation with end time"
  description = "test automation with end time"
  status = "enabled"
  end_time = "2024-12-31T23:59:59Z"
  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type = "object_set"
    })
  }
  sumo_logic_template = {
    external_config_id = "test-uuid"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "name", "test automation with end time"),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2.test", "end_time", "2024-12-31T23:59:59Z"),
				),
			},
		},
	})
}