package azure_devops_template_test

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccDataSourceConfig = orcasecurity.TestProviderConfig + `
data "orcasecurity_azure_devops_template" "test" {
  name = "tf: example"
}
`

func TestAccAzureDevopsTemplateDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_azure_devops_template.test", "name", "tf: example"),
					resource.TestCheckResourceAttrWith("data.orcasecurity_azure_devops_template.test", "id", func(value string) error {
						// it must be a valid UUID
						_, err := uuid.Parse(value)
						return err
					}),
				),
			},
		},
	})

}
