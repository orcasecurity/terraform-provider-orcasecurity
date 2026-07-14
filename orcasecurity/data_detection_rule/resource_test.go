package data_detection_rule_test

import (
	"regexp"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataDetectionRuleResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create (rule scoped by tags; wired to a fresh policy)
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_dspm_policy" "for_rule" {
  name        = "tf-acc-policy-for-rule"
  description = "policy for rule acceptance test"
  document = {
    detectors = ["*"]
  }
}

resource "orcasecurity_data_detection_rule" "test" {
  name     = "tf-acc-detection-rule"
  policies = [orcasecurity_dspm_policy.for_rule.id]
  tags     = [{ keys = ["*"], values = ["tf-acc"] }]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "name", "tf-acc-detection-rule"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "enabled", "false"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "action", "scan"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "feature", "DSPM Scanning"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "is_default_rule", "false"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "policies.#", "1"),
					resource.TestCheckResourceAttrSet("orcasecurity_data_detection_rule.test", "priority"),
					resource.TestCheckResourceAttrSet("orcasecurity_data_detection_rule.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_data_detection_rule.test", "organization_id"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_data_detection_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update: rename, enable, switch action
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_dspm_policy" "for_rule" {
  name        = "tf-acc-policy-for-rule"
  description = "policy for rule acceptance test"
  document = {
    detectors = ["*"]
  }
}

resource "orcasecurity_data_detection_rule" "test" {
  name     = "tf-acc-detection-rule-renamed"
  enabled  = true
  action   = "do_not_scan"
  policies = [orcasecurity_dspm_policy.for_rule.id]
  tags     = [{ keys = ["*"], values = ["tf-acc", "tf-acc-updated"] }]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "name", "tf-acc-detection-rule-renamed"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "enabled", "true"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "action", "do_not_scan"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "tags.0.values.#", "2"),
				),
			},
		},
	})
}

func TestAccDataDetectionRuleResource_ScopeRequired(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_data_detection_rule" "invalid" {
  name     = "tf-acc-rule-no-scope"
  policies = ["00000000-0000-0000-0000-000000000000"]
}
`,
				ExpectError: regexp.MustCompile(`Missing Attribute Configuration`),
			},
		},
	})
}

// A rule must attach at least one policy, matching the Orca UI (the API does
// not enforce this, so the provider does).
func TestAccDataDetectionRuleResource_PoliciesRequired(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_data_detection_rule" "no_policies" {
  name = "tf-acc-rule-no-policies"
  tags = [{ keys = ["*"], values = ["tf-acc"] }]
}
`,
				ExpectError: regexp.MustCompile(`The argument "policies" is required`),
			},
		},
	})
}
