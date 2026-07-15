package dspm_policy_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDspmPolicyResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_dspm_policy" "test" {
  name        = "tf-acc-dspm-policy"
  description = "test description"
  document = {
    detectors  = ["*"]
    categories = ["PII"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "name", "tf-acc-dspm-policy"),
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "description", "test description"),
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "feature", "DSPM Scanning"),
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "document.detectors.#", "1"),
					resource.TestCheckTypeSetElemAttr("orcasecurity_dspm_policy.test", "document.detectors.*", "*"),
					resource.TestCheckTypeSetElemAttr("orcasecurity_dspm_policy.test", "document.categories.*", "PII"),
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "is_default_policy", "false"),
					resource.TestCheckResourceAttrSet("orcasecurity_dspm_policy.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_dspm_policy.test", "organization_id"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_dspm_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_dspm_policy" "test" {
  name        = "tf-acc-dspm-policy-renamed"
  description = "test description updated"
  tags        = ["team:security"]
  document = {
    detectors = ["*"]
    regions   = ["Europe"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "name", "tf-acc-dspm-policy-renamed"),
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "tags.0", "team:security"),
					resource.TestCheckTypeSetElemAttr("orcasecurity_dspm_policy.test", "document.regions.*", "Europe"),
					resource.TestCheckNoResourceAttr("orcasecurity_dspm_policy.test", "document.categories"),
				),
			},
			// multi-element document, deliberately NOT in server-sorted order:
			// the server sorts every policy_document list on save, so this step
			// guards against a perpetual plan diff on order-sensitive attributes
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_dspm_policy" "test" {
  name        = "tf-acc-dspm-policy-renamed"
  description = "test description updated"
  tags        = ["team:security"]
  document = {
    detectors  = ["AUS_TAX_NUMBER", "AUSTRIA_TIN"]
    categories = ["PII", "PCI"]
    regions    = ["North America", "Europe"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "document.detectors.#", "2"),
					resource.TestCheckTypeSetElemAttr("orcasecurity_dspm_policy.test", "document.detectors.*", "AUS_TAX_NUMBER"),
					resource.TestCheckTypeSetElemAttr("orcasecurity_dspm_policy.test", "document.detectors.*", "AUSTRIA_TIN"),
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "document.categories.#", "2"),
					resource.TestCheckTypeSetElemAttr("orcasecurity_dspm_policy.test", "document.categories.*", "PCI"),
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "document.regions.#", "2"),
				),
			},
			// same config again: plan must be empty even though the server
			// stores the lists sorted
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_dspm_policy" "test" {
  name        = "tf-acc-dspm-policy-renamed"
  description = "test description updated"
  tags        = ["team:security"]
  document = {
    detectors  = ["AUS_TAX_NUMBER", "AUSTRIA_TIN"]
    categories = ["PII", "PCI"]
    regions    = ["North America", "Europe"]
  }
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
