package monday_resource_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The test provisions its own Monday credentials resource and looks it up by name through the
// data source (the reference creates an implicit dependency), so nothing pre-existing in the
// lab is required and everything created is destroyed automatically when the test finishes.
//
// The Orca backend validates the Monday API token against Monday.com at create time and
// rejects invalid tokens, so a real token is required. Provide it via
// ORCASECURITY_TEST_MONDAY_API_TOKEN; without it the test skips.
func TestAccMondayResourceDataSource(t *testing.T) {
	token := os.Getenv("ORCASECURITY_TEST_MONDAY_API_TOKEN")
	if token == "" {
		t.Skip("set ORCASECURITY_TEST_MONDAY_API_TOKEN (a valid Monday.com API token) to run: " +
			"the Orca backend validates the token against Monday.com at create time")
	}

	name := "tf-acc-test-monday-ds-" + uuid.NewString()[:8]
	config := orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_integration_monday_resource" "test" {
  name      = "%s"
  api_token = "%s"
}

data "orcasecurity_integration_monday_resource" "test" {
  name = orcasecurity_integration_monday_resource.test.name
}
`, name, token)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_integration_monday_resource.test", "name", name),
					resource.TestCheckResourceAttrWith("data.orcasecurity_integration_monday_resource.test", "id", func(value string) error {
						// the resource id must be a valid UUID
						_, err := uuid.Parse(value)
						return err
					}),
					resource.TestCheckResourceAttrSet("data.orcasecurity_integration_monday_resource.test", "account_slug"),
				),
			},
		},
	})
}
