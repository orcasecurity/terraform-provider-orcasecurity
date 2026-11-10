package createrecovery_test

import (
	"fmt"
	"net/http"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func init() {
	accCases = append(accCases, accCase{
		name: "data_detection_rule", address: "orcasecurity_data_detection_rule.t",
		method: http.MethodPut, post: "/api/scan_configuration/rules",
		config: func(n string) string {
			return fmt.Sprintf(`
resource "orcasecurity_dspm_policy" "for_rule" {
  name        = "%[1]s-policy"
  description = "WASP-1876 ambiguous-create acceptance test"
  document = {
    detectors = ["*"]
  }
}

resource "orcasecurity_data_detection_rule" "t" {
  name     = %[1]q
  enabled  = false
  policies = [orcasecurity_dspm_policy.for_rule.id]
  tags     = [{ keys = ["tf-acc-wasp1876"], values = [%[1]q] }]
}`, n)
		},
		ids: (*api_client.APIClient).ListDataDetectionRuleIDsByName,
	})
}
