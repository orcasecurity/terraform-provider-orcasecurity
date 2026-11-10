package createrecovery_test

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

const accAllowedReposTemplate = "k8sallowedrepos"

func accAdmissionControl(n string) string {
	return fmt.Sprintf(`
data "orcasecurity_admission_controller_template" "repos" {
  name = %q
}

resource "orcasecurity_admission_controller_control" "dep" {
  name        = %q
  description = "WASP-1876 ambiguous-create acceptance test"
  template_id = data.orcasecurity_admission_controller_template.repos.id
  cluster_scope = {
    kinds = [{
      kinds      = ["Pod"]
      api_groups = [""]
      versions   = [""]
    }]
  }
  input_parameters = jsonencode({ repos = ["registry.example.com"] })
}`, accAllowedReposTemplate, n)
}

func accAdmissionPolicy(n, control string) string {
	return fmt.Sprintf(`
resource "orcasecurity_admission_controller_policy" "dep" {
  name               = %q
  description        = "WASP-1876 ambiguous-create acceptance test"
  is_active          = false
  enforcement_action = "monitor"
  controls           = [%s]
}`, n, control)
}

func accAllowedReposTemplateID(c *api_client.APIClient) (string, error) {
	templates, err := c.GetAdmissionControllerTemplates()
	if err != nil {
		return "", err
	}
	for _, t := range templates {
		if t.Name == accAllowedReposTemplate {
			return t.ID, nil
		}
	}
	return "", fmt.Errorf("admission controller template %q not found", accAllowedReposTemplate)
}

func init() {
	accCases = append(accCases,
		accCase{
			name: "custom_dashboard", address: "orcasecurity_custom_dashboard.t", post: "/api/user_preferences",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_dashboard" "t" {
  name               = %q
  filter_data        = {}
  organization_level = false
  view_type          = "dashboard"
  extra_params = {
    description    = "WASP-1876 ambiguous-create acceptance test"
    version        = 2
    widgets_config = [{ id = "cloud-accounts-inventory", size = "sm" }]
  }
}`, n)
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.CustomDashboardCreateOp(api_client.CustomDashboard{Name: n, ViewType: "dashboard"}).IDs()
			},
		},
		accCase{
			name: "custom_widget", address: "orcasecurity_custom_widget.t", post: "/api/user_preferences",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_custom_widget" "t" {
  name               = %q
  organization_level = false
  extra_params = {
    type                = "donut"
    empty_state_message = "No data found"
    default_size        = "sm"
    is_new              = true
    subtitle            = "WASP-1876"
    description         = "WASP-1876 ambiguous-create acceptance test"
    settings = {
      field = {
        name = "Type"
        type = "str"
      }
      request_params = {
        query    = jsonencode({ models = ["AwsS3Bucket"], type = "object_set" })
        group_by = ["Type"]
      }
    }
  }
}`, n)
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.CustomWidgetCreateOp(api_client.CustomWidget{Name: n, ViewType: api_client.CustomWidgetViewType}).IDs()
			},
		},
		accCase{
			name: "shift_left_project", address: "orcasecurity_shift_left_project.t", post: "/api/shiftleft/projects/",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_shift_left_project" "t" {
  name             = %[1]q
  description      = "WASP-1876 ambiguous-create acceptance test"
  key              = %[1]q
  default_policies = true
}`, n)
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.ListShiftLeftProjectIDsByKey(n, n)
			},
		},
		accCase{
			name: "dspm_policy", address: "orcasecurity_dspm_policy.t", post: "/api/scan_configuration/dspm_policies",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_dspm_policy" "t" {
  name        = %q
  description = "WASP-1876 ambiguous-create acceptance test"
  document = {
    detectors = ["PII-Email"]
  }
}`, n)
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.ListDSPMPolicyIDsByName(n, "")
			},
		},
		accCase{
			name: "sensitive_data_identifier", address: "orcasecurity_sensitive_data_identifier.t", post: "/api/scan_configuration/dspm_detector",
			config: func(n string) string {
				return fmt.Sprintf(`
resource "orcasecurity_sensitive_data_identifier" "t" {
  title        = %q
  details      = "WASP-1876 ambiguous-create acceptance test"
  category     = "SECRET"
  sub_category = "API Keys and Tokens"
  enabled      = false
  properties = {
    conditions = [
      { value = "\\b(?P<secret>wasp1876_[a-f0-9]{32})\\b" }
    ]
  }
}`, n)
			},
			ids: (*api_client.APIClient).ListDSPMDetectorIDsByTitle,
		},
		accCase{
			name: "admission_controller_control", address: "orcasecurity_admission_controller_control.dep", post: "/api/admission_controller/controls",
			config: accAdmissionControl,
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				templateID, err := accAllowedReposTemplateID(c)
				if err != nil {
					return nil, err
				}
				return c.AdmissionControllerControlCreateOp(api_client.AdmissionControllerControl{Name: n, TemplateID: templateID}).IDs()
			},
		},
		accCase{
			name: "admission_controller_policy", address: "orcasecurity_admission_controller_policy.dep", post: "/api/admission_controller/policies",
			config: func(n string) string {
				return accAdmissionControl(n+"-control") +
					accAdmissionPolicy(n, "orcasecurity_admission_controller_control.dep.id")
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.AdmissionControllerPolicyCreateOp(api_client.AdmissionControllerPolicy{Name: n}).IDs()
			},
		},
		// A scope needs real clusters/accounts or full_organization; the inactive monitor-only policy keeps it inert.
		accCase{
			name: "admission_controller_policy_assignment", address: "orcasecurity_admission_controller_policy_assignment.t", post: "/api/admission_controller/scopes",
			config: func(n string) string {
				return accAdmissionControl(n+"-control") +
					accAdmissionPolicy(n+"-policy", "orcasecurity_admission_controller_control.dep.id") + fmt.Sprintf(`

resource "orcasecurity_admission_controller_policy_assignment" "t" {
  name              = %q
  description       = "WASP-1876 ambiguous-create acceptance test"
  full_organization = true
  policy_ids        = [orcasecurity_admission_controller_policy.dep.id]
}`, n)
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.AdmissionControllerScopeCreateOp(api_client.AdmissionControllerScope{Name: n}).IDs()
			},
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
