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
	ResourceType   = "orcasecurity_integration_webhook_template"
	Resource       = "terraformTestDataSourceInOrca"
	OrcaObject     = "terraformTestDataSourceInOrca"
)

// The data source is read by name, so the test provisions its own webhook and
// looks it up by that name. Referencing the resource's template_name in the
// data source creates an implicit dependency, so the webhook exists before the
// lookup runs and is torn down automatically when the test finishes.
var testAccDataSourceConfig = orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
  template_name = "%s"

  config = {
    webhook_url = "https://example.com/orca-tf-acc-test"
    type        = "common"
  }
}

data "%s" "%s" {
  name = %s.%s.template_name
}
`, ResourceType, Resource, OrcaObject, DataSourceType, DataSource, ResourceType, Resource)

func TestAccWebhookDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("data.%s.%s", DataSourceType, DataSource), "name", OrcaObject),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.%s.%s", DataSourceType, DataSource), "config.type", "common"),
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
