package group_test

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccGroupDataSourceConfig = orcasecurity.TestProviderConfig + `
data "orcasecurity_group" "test" {
	name = "Login Only - No Visibility"
}
`

func TestAccGroupDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_group.test", "name", "Login Only - No Visibility"),
					resource.TestCheckResourceAttrWith("data.orcasecurity_group.test", "id", func(value string) error {
						// it must be a valid UUID
						_, err := uuid.Parse(value)
						return err
					}),
					resource.TestCheckResourceAttrSet("data.orcasecurity_group.test", "description"),
					resource.TestCheckResourceAttrSet("data.orcasecurity_group.test", "sso_group"),
				),
			},
		},
	})
}
