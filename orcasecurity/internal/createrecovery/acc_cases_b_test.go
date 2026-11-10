package createrecovery_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func inviteEmail(name string) string { return name + "@example.com" }

func fakeAWSAccountID(name string) string {
	n, _ := strconv.Atoi(name[strings.LastIndex(name, "-")+1:])
	return fmt.Sprintf("%012d", n)
}

func groupAccessIDs(c *api_client.APIClient, name string) (map[string]bool, error) {
	groups, err := c.ListGroupIDsByName(name)
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for gid := range groups {
		rows, err := c.ListGroupAccessForGroup(gid)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			ids[r.ID] = true
		}
	}
	return ids, nil
}

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
			name: "custom_discovery_alert", address: "orcasecurity_custom_discovery_alert.t", post: "/api/sonar/rules",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_discovery_alert" "t" {
  name          = %q
  description   = "WASP-1876 ambiguous-create acceptance test"
  rule_json     = jsonencode({ "models" : ["AwsS3Bucket"], "type" : "object_set", "with" : { "key" : "Name", "type" : "str", "values" : [%q], "operator" : "in" } })
  orca_score    = 1
  category      = "Best practices"
  context_score = false
  remediation_text = {
    enable = true
    text   = "WASP-1876 acc remediation"
  }
}`, n, n)
			},
			ids: (*api_client.APIClient).ListCustomSonarRuleIDsByName,
		},
		accCase{
			name: "automation_v2", address: "orcasecurity_automation_v2.t", post: "/api/automations",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_automation_v2" "t" {
  name        = %q
  description = "WASP-1876 ambiguous-create acceptance test"
  status      = "disabled"
  filter = {
    sonar_query = jsonencode({ "models" : ["Alert"], "type" : "object_set", "with" : { "key" : "AlertType", "type" : "str", "values" : [%q], "operator" : "in" } })
  }
  email_template = {
    email = [%q]
  }
}`, n, n, inviteEmail(n))
			},
			ids: (*api_client.APIClient).ListAutomationV2IDsByName,
		},
		accCase{
			name: "user_invite", address: "orcasecurity_add_users.t", post: "/api/user_invites/bulk_create/",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_role" "r" {
  name              = %q
  description       = "WASP-1876 ambiguous-create acceptance test"
  permission_groups = ["assets.asset.read"]
}

resource "orcasecurity_add_users" "t" {
  email             = %q
  role_id           = orcasecurity_custom_role.r.id
  should_send_email = false
}`, n, inviteEmail(n))
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.ListUserInviteIDsByEmail(inviteEmail(n))
			},
		},
		accCase{
			name: "trusted_cloud_account", address: "orcasecurity_trusted_cloud_account.t", post: "/api/organization/trusted_accounts",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_trusted_cloud_account" "t" {
  account_name      = %q
  description       = "WASP-1876 ambiguous-create acceptance test (fake account id)"
  cloud_provider    = "aws"
  cloud_provider_id = %q
}`, n, fakeAWSAccountID(n))
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.ListTrustedCloudAccountIDs("aws", fakeAWSAccountID(n))
			},
		},
		accCase{
			name: "group_access", address: "orcasecurity_group_access.t", post: "/api/rbac/access/group",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_role" "r" {
  name              = %q
  description       = "WASP-1876 ambiguous-create acceptance test"
  permission_groups = ["assets.asset.read"]
}

resource "orcasecurity_group" "g" {
  name        = %q
  description = "WASP-1876 ambiguous-create acceptance test"
  sso_group   = false
}

resource "orcasecurity_group_access" "t" {
  group_id           = orcasecurity_group.g.id
  role_id            = orcasecurity_custom_role.r.id
  all_cloud_accounts = true
}`, n, n)
			},
			ids: groupAccessIDs,
		},
		accCase{
			name: "external_service_config", address: "orcasecurity_integration_opsgenie.t", post: "/api/external_service/config",
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
			name: "jira_cloud_template", address: "orcasecurity_integration_jira_cloud_template.t", post: "/api/external_service/config",
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
