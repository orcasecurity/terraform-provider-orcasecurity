package createrecovery_test

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func init() {
	accCases = append(accCases,
		accCase{
			name: "custom_tag_rule", address: "orcasecurity_custom_tag_rule.t", post: "/api/custom_tags",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_tag_rule" "t" {
  name    = %q
  rule     = "AwsEc2Instance with PublicIps"
  disabled = true
  tags     = { "wasp-1876" = "acc" }
}`, n)
			},
			ids: (*api_client.APIClient).ListCustomTagRuleIDsByName,
		},
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
			name: "custom_compliance_framework", address: "orcasecurity_custom_compliance_framework.t", post: "/api/compliance/frameworks",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_compliance_framework" "t" {
  name        = %q
  description = "WASP-1876 ambiguous-create acceptance test"
  sections = [{
    name  = "Section 1"
    tests = [{ rule_id = "rc7bcf3b77f", rule_id_in_framework = "1.1" }]
  }]
}`, n)
			},
			ids: (*api_client.APIClient).ListCustomComplianceFrameworkIDsByName,
		},
		accCase{
			name: "scheduled_report", address: "orcasecurity_scheduled_report.t", post: "/api/reporting/scheduled_reports",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_scheduled_report" "t" {
  name              = %q
  type              = "discovery_svl"
  format            = "csv"
  recurrence        = "monthly"
  first_report_date = "2036-01-01T00:00:00Z"
  export_time       = "00:00:00"
  status            = "disabled"
  recipients_emails = ["wasp-1876@example.com"]
  sonar_query       = jsonencode({ models = ["AwsS3Bucket"], type = "object_set" })
}`, n)
			},
			ids: (*api_client.APIClient).ListScheduledReportIDsByName,
		},
	)
}
