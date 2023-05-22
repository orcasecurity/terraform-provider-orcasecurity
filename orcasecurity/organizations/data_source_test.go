package organizations_test

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccDataSourceConfig = orcasecurity.TestProviderConfig + `
data "orcasecurity_organization" "test" {
  
}
`

func TestAccJiraTemplateDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_organization.test", "name", "OTS"),
					resource.TestCheckResourceAttrWith("data.orcasecurity_organization.test", "id", func(value string) error {
						_, err := uuid.Parse(value)
						return err
					}),
				),
			},
		},
	})
}
