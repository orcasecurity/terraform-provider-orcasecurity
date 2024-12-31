package webhook_test

import (
<<<<<<< HEAD
=======
	"fmt"
>>>>>>> alert-docs-update
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

<<<<<<< HEAD
// Requires that webhook named "tf_test" exists on the API side
const testAccDataSourceConfig = orcasecurity.TestProviderConfig + `
data "orcasecurity_webhook" "test" {
  name = "tf_test"
}
`
=======
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
>>>>>>> alert-docs-update

func TestAccWebhookDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
<<<<<<< HEAD
					resource.TestCheckResourceAttr("data.orcasecurity_webhook.test", "name", "tf_test"),
					resource.TestCheckResourceAttrWith("data.orcasecurity_webhook.test", "id", func(value string) error {
=======
					resource.TestCheckResourceAttr(fmt.Sprintf("data.%s.%s", DataSourceType, DataSource), "name", fmt.Sprintf("%s", OrcaObject)),
					resource.TestCheckResourceAttrWith(fmt.Sprintf("data.%s.%s", DataSourceType, DataSource), "id", func(value string) error {
>>>>>>> alert-docs-update
						// it must be a valid UUID
						_, err := uuid.Parse(value)
						return err
					}),
				),
			},
		},
	})
}
