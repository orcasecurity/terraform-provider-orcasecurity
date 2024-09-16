package business_unit_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBusinessUnitResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_business_unit" "business_unit_for_aws" {
    name = "AWS"
    filter_data = {
        cloud_provider = ["aws"]
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_business_unit.business_unit_for_aws", "name", "AWS"),
					resource.TestCheckResourceAttr("orcasecurity_business_unit.business_unit_for_aws", "filter_data.cloud_provider[0]", "aws"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_business_unit.business_unit_for_aws",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_business_unit" "business_unit_for_azure" {
    name = "Azure"
    filter_data = {
        cloud_provider = ["azure"]
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_business_unit.business_unit_for_azure", "name", "Azure"),
					resource.TestCheckResourceAttr("orcasecurity_business_unit.business_unit_for_azure", "filter_data.cloud_provider[0]", "azure"),
				),
			},
		},
	})
}
