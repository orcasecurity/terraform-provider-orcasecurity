package azure_devops_template_test

import (
	"fmt"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccDataSourceConfig(name string) string {
	return orcasecurity.TestProviderConfig + fmt.Sprintf(`
data "orcasecurity_azure_devops_template" "test" {
  name = %q
}
`, name)
}

func TestAccAzureDevopsTemplateDataSource(t *testing.T) {
	name := acctest.ExistingTemplateName(t, "azure_devops", "ORCASECURITY_ACC_AZURE_DEVOPS_TEMPLATE_NAME")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_azure_devops_template.test", "name", name),
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
