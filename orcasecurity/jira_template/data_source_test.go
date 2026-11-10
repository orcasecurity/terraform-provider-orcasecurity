package jira_template_test

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
data "orcasecurity_jira_template" "test" {
  name = %q
}
`, name)
}

func TestAccJiraTemplateDataSource(t *testing.T) {
	name := acctest.ExistingTemplateName(t, "jira", "ORCASECURITY_ACC_JIRA_TEMPLATE_NAME")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_jira_template.test", "name", name),
					resource.TestCheckResourceAttrWith("data.orcasecurity_jira_template.test", "id", func(value string) error {
						// it must be a valid UUID
						_, err := uuid.Parse(value)
						return err
					}),
				),
			},
		},
	})
}
