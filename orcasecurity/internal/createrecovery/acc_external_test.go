package createrecovery_test

import (
	"fmt"
	"os"
	"sync"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func configIDs(service string) func(*api_client.APIClient, string) (map[string]bool, error) {
	return func(c *api_client.APIClient, name string) (map[string]bool, error) {
		return api_client.ListExternalServiceConfigIDs(c, service, name)
	}
}

// The create validates mapping against a live Jira project, so borrow site/project/issue type from an existing template.
const jiraTemplateEnv = "ORCASECURITY_ACC_JIRA_CLOUD_TEMPLATE"

var jiraSource = sync.OnceValue(func() *api_client.JiraCloudTemplate {
	endpoint, token := os.Getenv("ORCASECURITY_API_ENDPOINT"), os.Getenv("ORCASECURITY_API_TOKEN")
	c, err := api_client.NewAPIClient(&endpoint, &token)
	if err != nil {
		panic(err)
	}
	tpl, err := c.GetJiraCloudTemplate(os.Getenv(jiraTemplateEnv))
	if err != nil || tpl == nil {
		panic(fmt.Sprintf("%s=%q must name an existing Jira Cloud template (err: %v)", jiraTemplateEnv, os.Getenv(jiraTemplateEnv), err))
	}
	return tpl
})

func init() {
	accCases = append(accCases,
		accCase{
			name: "external_service_config", address: "orcasecurity_integration_opsgenie.t", path: "/api/external_service/config",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_integration_opsgenie" "t" {
  template_name = %q
  opsgenie_key  = "00000000-0000-0000-0000-000000000000"
  is_enabled    = false
}`, n)
			},
			ids: configIDs(api_client.OpsgenieServiceName),
		},
	)
	if os.Getenv(jiraTemplateEnv) != "" {
		accCases = append(accCases, accCase{
			name: "jira_cloud_template", address: "orcasecurity_integration_jira_cloud_template.t", path: "/api/external_service/config",
			config: func(n string) string {
				src := jiraSource()
				return fmt.Sprintf(`
resource "orcasecurity_integration_jira_cloud_template" "t" {
  template_name = %q
  resource_id   = %q
  resource_url  = %q
  project_id    = %q
  issue_type_id = %q
  mapping_json  = jsonencode({ summary = ["alert_name"], description = ["details"] })
  is_enabled    = false
}`, n, src.Config.ResourceID, src.Config.ResourceURL, src.Config.ProjectID, src.Config.IssueTypeID)
			},
			ids: configIDs(api_client.JiraCloudServiceName),
		})
	}
}
