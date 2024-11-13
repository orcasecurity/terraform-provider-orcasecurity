package trusted_cloud_account_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestTrustedCloudAccountResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_trusted_cloud_account" "Orca_TF_Provider_Acceptance_Test_Account" {
  account_name      = "Orca TF Provider Acceptance Test Account"
  description       = "Dummy Description"
  cloud_provider    = "aws"
  cloud_provider_id = "12341234123445678912"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_trusted_cloud_account.Orca_TF_Provider_Acceptance_Test_Account", "name", "Orca TF Provider Acceptance Test Account"),
					resource.TestCheckResourceAttr("orcasecurity_trusted_cloud_account.Orca_TF_Provider_Acceptance_Test_Account", "description", "Dummy Description"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_trusted_cloud_account.account-1",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_trusted_cloud_account" "account-1" {
  account_name      = "test44912"
  description       = "test2"
  cloud_provider    = "aws"
  cloud_provider_id = "12341234123445678912"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_trusted_cloud_account.account-1", "name", "test44912"),
					resource.TestCheckResourceAttr("orcasecurity_trusted_cloud_account.account-1", "description", "test2"),
				),
			},
		},
	})
}
