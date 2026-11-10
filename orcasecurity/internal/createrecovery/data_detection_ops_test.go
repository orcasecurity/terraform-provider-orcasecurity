package createrecovery_test

import (
	"fmt"
	"net/http"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func init() {
	base := "/api/scan_configuration/rules"
	row := func(id string) string {
		return fmt.Sprintf(`{"rule_id":%q,"rule_name":%q,"feature":"DSPM Scanning","action":"scan"}`, id, stubTag)
	}
	opCases = append(opCases, opCase{
		name:   "data_detection_rule",
		method: http.MethodPut,
		post:   base,
		get: map[string]func(bool) string{
			base: func(created bool) string {
				return rows(created, row, func(items string) string { return "[" + items + "]" })
			},
			base + "/" + newID: func(bool) string { return `{"status":"success","data":` + row(newID) + `}` },
		},
		empty: `[]`,
		op: func(c *api_client.APIClient) runner {
			return runOp(c.DataDetectionRuleCreateOp(api_client.DataDetectionRule{Name: stubTag, Feature: "DSPM Scanning", Action: "scan"}))
		},
	})
}
