package createrecovery_test

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func listedByName(created bool, idKey, name string, wrap func(string) string) string {
	return rows(created, func(id string) string {
		return fmt.Sprintf(`{%q:%q,"name":%q}`, idKey, id, name)
	}, wrap)
}

// idNameCase is a create whose list rows and by-id read are {"id","name"} objects under "data".
func idNameCase(name, path, listedName string, op func(c *api_client.APIClient) runner) opCase {
	return opCase{
		name: name,
		post: path,
		get: map[string]func(bool) string{
			path: func(created bool) string { return listedByName(created, "id", listedName, dataList) },
			path + "/" + newID: func(bool) string {
				return fmt.Sprintf(`{"data":{"id":%q,"name":%q}}`, newID, listedName)
			},
		},
		op: op,
	}
}

func withAnswer(c opCase, answer string) opCase {
	c.answer = answer
	return c
}

func sonarRuleCase(name string, op func(c *api_client.APIClient) runner) opCase {
	return opCase{
		name: name,
		post: "/api/sonar/rules",
		get: map[string]func(bool) string{
			"/api/sonar/rules/custom": func(created bool) string { return listedByName(created, "rule_id", stubTag, dataList) },
			"/api/sonar/rules/" + newID: func(bool) string {
				return fmt.Sprintf(`{"data":{"rule_id":%q,"name":%q,"rule_type":"t1"}}`, newID, stubTag)
			},
			"/api/alerts/custom_remediation_text": func(bool) string { return `{"alert_type":"t1","enabled":false,"custom_text":""}` },
		},
		op: op,
	}
}

func init() {
	opCases = append(opCases,
		sonarRuleCase("custom_sonar_alert", func(c *api_client.APIClient) runner {
			return runOp(c.CustomSonarAlertCreateOp(api_client.CustomAlert{Name: stubTag}))
		}),
		sonarRuleCase("custom_discovery_alert", func(c *api_client.APIClient) runner {
			return runOp(c.CustomDiscoveryAlertCreateOp(api_client.CustomDiscoveryAlert{Name: stubTag}))
		}),
		idNameCase("custom_tag_rule", "/api/custom_tags", stubTag, func(c *api_client.APIClient) runner {
			return runOp(c.CustomTagRuleCreateOp(api_client.CustomTagRule{Name: stubTag, RuleType: api_client.CustomTagRuleRuleTypeString}))
		}),
		withAnswer(idNameCase("custom_tag_rule_without_id", "/api/custom_tags", stubTag, func(c *api_client.APIClient) runner {
			return runOp(c.CustomTagRuleCreateOp(api_client.CustomTagRule{Name: stubTag, RuleType: api_client.CustomTagRuleRuleTypeString}))
		}), `{"data":{},"status":"success"}`),
		idNameCase("automation_v2", "/api/automations", stubTag, func(c *api_client.APIClient) runner {
			return runOp(c.AutomationV2CreateOp(api_client.AutomationV2{Name: stubTag}, true))
		}),
		idNameCase("scheduled_report", "/api/reporting/scheduled_reports", "tf:recovery", func(c *api_client.APIClient) runner {
			return runOp(c.ScheduledReportCreateOp(api_client.ScheduledReport{Name: "tf/recovery"}))
		}),
		opCase{
			name: "custom_compliance_framework",
			post: "/api/compliance/frameworks",
			get: map[string]func(bool) string{
				"/api/compliance/frameworks/select": func(created bool) string {
					return rows(created, func(id string) string {
						return fmt.Sprintf(`%q:{"display_name":" TF-Recovery ","custom":true}`, id)
					}, func(items string) string {
						return `{"33333333":{"display_name":"tf-recovery","custom":false},` + items + `}`
					})
				},
				"/api/compliance/frameworks/" + newID: func(bool) string {
					return fmt.Sprintf(`{"data":{"id":%q,"display_name":%q,"custom":true}}`, newID, stubTag)
				},
			},
			op: func(c *api_client.APIClient) runner {
				return runOp(c.CustomComplianceFrameworkCreateOp(api_client.CustomComplianceFrameworkRequest{Name: stubTag}))
			},
		},
	)
}
