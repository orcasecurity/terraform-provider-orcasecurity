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
			name: "dspm_policy", address: "orcasecurity_dspm_policy.t", path: "/api/scan_configuration/dspm_policies",
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
			name: "sensitive_data_identifier", address: "orcasecurity_sensitive_data_identifier.t", path: "/api/scan_configuration/dspm_detector",
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
			name: "admission_controller_control", address: "orcasecurity_admission_controller_control.dep", path: "/api/admission_controller/controls",
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
			name: "admission_controller_policy", address: "orcasecurity_admission_controller_policy.dep", path: "/api/admission_controller/policies",
			config: func(n string) string {
				return accAdmissionControl(n+"-control") +
					accAdmissionPolicy(n, "orcasecurity_admission_controller_control.dep.id")
			},
			ids: func(c *api_client.APIClient, n string) (map[string]bool, error) {
				return c.AdmissionControllerPolicyCreateOp(api_client.AdmissionControllerPolicy{Name: n}).IDs()
			},
		},
		accCase{
			name: "admission_controller_policy_assignment", address: "orcasecurity_admission_controller_policy_assignment.t", path: "/api/admission_controller/scopes",
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
	)
}
