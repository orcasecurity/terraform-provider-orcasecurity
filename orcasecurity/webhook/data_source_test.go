package webhook_test

import (
	"fmt"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	DataSourceType = "orcasecurity_webhook"
	DataSource     = "terraformTestDataSource"
	OrcaObject     = "terraformTestDataSourceInOrca"
)

// Requires that a webhook with the proper object name exist within the Orca org that's being used as a test environment.
var testAccDataSourceConfig = orcasecurity.TestProviderConfig + fmt.Sprintf(`
data "%s" "%s" {
  name = "%s"
}
`, DataSourceType, DataSource, OrcaObject)

func TestAccWebhookDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("data.%s.%s", DataSourceType, DataSource), "name", fmt.Sprintf("%s", OrcaObject)),
					resource.TestCheckResourceAttrWith(fmt.Sprintf("data.%s.%s", DataSourceType, DataSource), "id", func(value string) error {
						// it must be a valid UUID
						_, err := uuid.Parse(value)
						return err
					}),
				),
			},
		},
	})
}
