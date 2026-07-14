package sensitive_data_identifier_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSensitiveDataIdentifierResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_sensitive_data_identifier" "test" {
  title        = "tf-acc-identifier"
  details      = "test details"
  category     = "PII"
  sub_category = "Personal"
  properties = {
    conditions = [
      { value = "[0-9]{9}" }
    ]
    sensitivity  = "high"
    significance = "major"
    keywords     = ["ssn"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "title", "tf-acc-identifier"),
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "category", "PII"),
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "enabled", "true"),
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "properties.conditions.0.source", "content"),
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "properties.conditions.0.operator", "match"),
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "properties.conditions.0.value", "[0-9]{9}"),
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "properties.sensitivity", "high"),
					resource.TestCheckResourceAttrSet("orcasecurity_sensitive_data_identifier.test", "properties.detection_types.#"),
					resource.TestCheckResourceAttrSet("orcasecurity_sensitive_data_identifier.test", "id"),
					resource.TestCheckResourceAttrSet("orcasecurity_sensitive_data_identifier.test", "organization_id"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_sensitive_data_identifier.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update: rename, disable, tighten detection types
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_sensitive_data_identifier" "test" {
  title        = "tf-acc-identifier-renamed"
  details      = "test details updated"
  category     = "PHI"
  sub_category = "Medical"
  enabled      = false
  properties = {
    conditions = [
      { value = "[0-9]{10}" }
    ]
    detection_types = ["text"]
    sensitivity     = "medium"
    significance    = "moderate"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "title", "tf-acc-identifier-renamed"),
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "category", "PHI"),
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "enabled", "false"),
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "properties.detection_types.#", "1"),
					resource.TestCheckResourceAttr("orcasecurity_sensitive_data_identifier.test", "properties.detection_types.0", "text"),
				),
			},
		},
	})
}
