package jira_cloud_resource_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Jira Cloud sites are connected via the OAuth flow in the Orca UI; /api/jira/resources is
// read-only, so the test cannot provision (or destroy) a site of its own. The site to read is
// therefore taken from ORCASECURITY_TEST_JIRA_CLOUD_SITE_NAME instead of being hardcoded;
// without it the test skips. It is a pure read — nothing is created or destroyed. Shape is
// asserted (valid UUID id, https:// url) rather than fragile exact values.
func TestAccJiraCloudResourceDataSource(t *testing.T) {
	name := os.Getenv("ORCASECURITY_TEST_JIRA_CLOUD_SITE_NAME")
	if name == "" {
		t.Skip("set ORCASECURITY_TEST_JIRA_CLOUD_SITE_NAME to the name of a connected Jira Cloud site to run: " +
			"sites are created via the OAuth flow in the Orca UI and cannot be provisioned by the API")
	}

	config := orcasecurity.TestProviderConfig + fmt.Sprintf(`
data "orcasecurity_integration_jira_cloud_resource" "test" {
  name = "%s"
}
`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_integration_jira_cloud_resource.test", "name", name),
					resource.TestCheckResourceAttrWith("data.orcasecurity_integration_jira_cloud_resource.test", "id", func(value string) error {
						// the cloud id must be a valid UUID
						_, err := uuid.Parse(value)
						return err
					}),
					resource.TestCheckResourceAttrWith("data.orcasecurity_integration_jira_cloud_resource.test", "url", func(value string) error {
						if !strings.HasPrefix(value, "https://") {
							return fmt.Errorf("expected an https:// url, got %q", value)
						}
						return nil
					}),
				),
			},
		},
	})
}
