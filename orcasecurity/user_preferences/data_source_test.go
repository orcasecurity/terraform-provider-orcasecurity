package user_preferences_test

import (
	"strconv"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccDataSourceConfig = orcasecurity.TestProviderConfig + `
data "orcasecurity_user_preferences" "test" {}
`

func TestAccUserPreferencesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("data.orcasecurity_user_preferences.test", "custom_widget_ids.#", func(value string) error {
						n, err := strconv.Atoi(value)
						if err != nil {
							return err
						}
						if n < 0 {
							return strconv.ErrSyntax
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("data.orcasecurity_user_preferences.test", "custom_widgets.#", func(value string) error {
						n, err := strconv.Atoi(value)
						if err != nil {
							return err
						}
						if n < 0 {
							return strconv.ErrSyntax
						}
						return nil
					}),
				),
			},
		},
	})
}
