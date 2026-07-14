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
  tags     = ["tf-acc"]
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
  name    = "tf-acc-detection-rule-renamed"
  enabled = true
  action  = "do_not_scan"
  tags    = ["tf-acc", "tf-acc-updated"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "name", "tf-acc-detection-rule-renamed"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "enabled", "true"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "action", "do_not_scan"),
					resource.TestCheckResourceAttr("orcasecurity_data_detection_rule.test", "tags.#", "2"),
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
  name = "tf-acc-rule-no-scope"
}
`,
				ExpectError: regexp.MustCompile(`At least one attribute out of`),
			},
		},
	})
}
