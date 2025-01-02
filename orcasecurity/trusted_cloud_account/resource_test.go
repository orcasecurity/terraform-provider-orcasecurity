package trusted_cloud_account_test

import (
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	ResourceType = "orcasecurity_trusted_cloud_account"
	Resource     = "terraformTestResource"
	OrcaObject   = "terraformTestResourceInOrca"
)

func TestTrustedCloudAccountResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
  account_name      = "%s"
  description       = "Dummy Description"
  cloud_provider    = "aws"
  cloud_provider_id = "12341234123445678912"
}
`, ResourceType, Resource, OrcaObject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "account_name", OrcaObject),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "description", "Dummy Description"),
				),
			},
			// import
			{
				ResourceName:      fmt.Sprintf("%s.%s", ResourceType, Resource),
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
				resource "%s" "%s" {
				  account_name      = "%s"
				  description       = "Dummy Description"
				  cloud_provider    = "aws"
				  cloud_provider_id = "12341234123445678912"
				}
				`, ResourceType, Resource, OrcaObject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "account_name", OrcaObject),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "description", "Dummy Description"),
				),
			},
		},
	})
}
