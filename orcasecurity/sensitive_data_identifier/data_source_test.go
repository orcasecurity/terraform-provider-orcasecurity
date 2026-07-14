package sensitive_data_identifier_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSensitiveDataIdentifiersDataSource_FilterByTitle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_sensitive_data_identifier" "seed" {
  title        = "tf-acc-ds-seed"
  details      = "seed identifier for data source test"
  category     = "PII"
  sub_category = "Personal"
  properties = {
    conditions = [
      { value = "[0-9]{9}" }
    ]
  }
}

data "orcasecurity_sensitive_data_identifiers" "by_title" {
  title = orcasecurity_sensitive_data_identifier.seed.title
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_sensitive_data_identifiers.by_title", "identifiers.#", "1"),
					resource.TestCheckResourceAttr("data.orcasecurity_sensitive_data_identifiers.by_title", "identifiers.0.title", "tf-acc-ds-seed"),
					resource.TestCheckResourceAttr("data.orcasecurity_sensitive_data_identifiers.by_title", "identifiers.0.category", "PII"),
					resource.TestCheckResourceAttr("data.orcasecurity_sensitive_data_identifiers.by_title", "identifiers.0.is_custom", "true"),
					resource.TestCheckResourceAttr("data.orcasecurity_sensitive_data_identifiers.by_title", "identifiers.0.enabled", "true"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_sensitive_data_identifiers.by_title", "identifiers.0.id"),
				),
			},
		},
	})
}
