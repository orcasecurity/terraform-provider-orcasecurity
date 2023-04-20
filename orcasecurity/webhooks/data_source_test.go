package webhooks_test

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Requires that webhook named "tf_test" exists on the API side
const testAccDataSourceConfig = orcasecurity.TestProviderConfig + `
data "orcasecurity_webhook" "test" {
  name = "tf_test"
}
`

func TestAccWebhookDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_webhook.test", "name", "tf_test"),
					resource.TestCheckResourceAttrWith("data.orcasecurity_webhook.test", "id", func(value string) error {
						// it must be a valid UUID
						_, err := uuid.Parse(value)
						return err
					}),
				),
			},
		},
	})
}
