package createrecovery_test

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func listAndGetCase(name, base string, row func(id string) string, op func(c *api_client.APIClient) runner) opCase {
	return opCase{
		name: name,
		post: base,
		get: map[string]func(bool) string{
			base:               func(created bool) string { return rows(created, row, dataList) },
			base + "/" + newID: func(bool) string { return `{"data":` + row(newID) + `}` },
		},
		op: op,
	}
}

func namedRow(format string) func(id string) string {
	return func(id string) string { return fmt.Sprintf(format, id, stubTag) }
}

func init() {
	opCases = append(opCases,
		listAndGetCase("dspm_policy", "/api/scan_configuration/dspm_policies",
			namedRow(`{"policy_id":%q,"policy_name":%q,"feature":"DSPM Scanning","tags":{},"policy_document":{"selector_detectors":[]}}`),
			func(c *api_client.APIClient) runner {
				return runOp(c.DSPMPolicyCreateOp(api_client.DSPMPolicy{Name: stubTag}))
			}),
		listAndGetCase("dspm_detector", "/api/scan_configuration/dspm_detector",
			namedRow(`{"id":%q,"title":%q,"is_custom":true}`),
			func(c *api_client.APIClient) runner {
				return runOp(c.DSPMDetectorCreateOp(api_client.DSPMDetector{Title: stubTag}))
			}),
		listAndGetCase("admission_controller_control", "/api/admission_controller/controls",
			namedRow(`{"id":%q,"name":%q,"template_id":"tpl-1"}`),
			func(c *api_client.APIClient) runner {
				return runOp(c.AdmissionControllerControlCreateOp(api_client.AdmissionControllerControl{Name: stubTag, TemplateID: "tpl-1"}))
			}),
		listAndGetCase("admission_controller_policy", "/api/admission_controller/policies",
			namedRow(`{"id":%q,"name":%q}`),
			func(c *api_client.APIClient) runner {
				return runOp(c.AdmissionControllerPolicyCreateOp(api_client.AdmissionControllerPolicy{Name: stubTag}))
			}),
		listAndGetCase("admission_controller_scope", "/api/admission_controller/scopes",
			namedRow(`{"id":%q,"name":%q}`),
			func(c *api_client.APIClient) runner {
				return runOp(c.AdmissionControllerScopeCreateOp(api_client.AdmissionControllerScope{Name: stubTag}))
			}),
	)
}
