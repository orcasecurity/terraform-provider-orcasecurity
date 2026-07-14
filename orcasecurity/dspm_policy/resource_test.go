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
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "document.detectors.0", "*"),
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "document.categories.0", "PII"),
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
    regions   = ["EU"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "name", "tf-acc-dspm-policy-renamed"),
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "tags.0", "team:security"),
					resource.TestCheckResourceAttr("orcasecurity_dspm_policy.test", "document.regions.0", "EU"),
					resource.TestCheckNoResourceAttr("orcasecurity_dspm_policy.test", "document.categories"),
				),
			},
		},
	})
}
