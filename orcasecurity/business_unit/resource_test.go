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
    name = "AWSBU"
    filter_data = {
        cloud_provider = ["aws"]
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_business_unit.business_unit_for_aws", "name", "AWSBU"),
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
    name = "AzureBU"
    filter_data = {
        cloud_provider = ["azure"]
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_business_unit.business_unit_for_azure", "name", "AzureBU"),
					resource.TestCheckResourceAttr("orcasecurity_business_unit.business_unit_for_azure", "filter_data.cloud_provider[0]", "azure"),
				),
			},
			/*{
				Config: "", // Empty config forces destroy of all resources
			},*/
		},
	})
}

func TestAccBusinessUnitResource_ShiftLeft(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_business_unit" "business_unit_for_sl" {
    name = "SL BU"
    shiftleft_filter_data = {
        shiftleft_project_id = ["577ba5de-3837-4db1-999f-dd2524e09e52"]
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_business_unit.business_unit_for_sl", "shiftleft_filter_data.shiftleft_project_id[0]", "577ba5de-3837-4db1-999f-dd2524e09e52"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_business_unit.business_unit_for_sl",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_business_unit" "business_unit_for_sl" {
    name = "AWS"
    shiftleft_filter_data = {
        shiftleft_project_id = ["577ba5de-3837-4db1-999f-dd2524e09e52"]
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_business_unit.business_unit_for_sl", "shiftleft_filter_data.shiftleft_project_id[0]", "577ba5de-3837-4db1-999f-dd2524e09e52"),
				),
			},
			/*{
				Config: "", // Empty config forces destroy of all resources
			},*/
		},
	})
}
