package business_unit_test

import (
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	ResourceType = "orcasecurity_business_unit"
	Resource     = "terraformTestResource"
	OrcaObject1  = "terraformTestResourceInOrcaAws"
	OrcaObject2  = "terraformTestResourceInOrcaAzure"
	OrcaObject3  = "terraformTestResourceInOrcaShiftLeftProjects"
)

func TestAccBusinessUnitResource_CloudProvider(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
    name = "%s"
    filter_data = {
        cloud_providers = ["aws"]
    }
}`, ResourceType, Resource, OrcaObject1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "name", OrcaObject1),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "filter_data.cloud_providers.0", "aws"),
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
    name = "%s"
    filter_data = {
        cloud_providers = ["azure"]
    }
}`, ResourceType, Resource, OrcaObject2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "name", OrcaObject2),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "filter_data.cloud_providers.0", "azure"),
				),
			},
		},
	})
}

func TestAccBusinessUnitResource_ShiftLeft(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
                resource "%s" "%s" {
                    name = "%s"
                    shiftleft_filter_data = {
                        shiftleft_project_ids = ["1"]
                    }
                }`, ResourceType, Resource, OrcaObject3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "shiftleft_filter_data.shiftleft_project_ids.0", "1"),
					// Add explicit checks for filter_data to be empty/null if that's what you expect
					resource.TestCheckNoResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "filter_data"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("%s.%s", ResourceType, Resource),
				ImportState:       true,
				ImportStateVerify: true,
			},
			// ... rest of the test
		},
	})
}

func TestAccBusinessUnitResource_CloudAndShiftLeft(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
                resource "%s" "%s" {
                    name = "%s"

					filter_data = {
                        cloud_account_ids = ["12341234"]
                    }

                    shiftleft_filter_data = {
                        shiftleft_project_ids = ["1"]
                    }
                }`, ResourceType, Resource, OrcaObject3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "shiftleft_filter_data.shiftleft_project_ids.0", "1"),
					// Add explicit checks for filter_data to be empty/null if that's what you expect
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "filter_data.cloud_account_ids.0", "12341234"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("%s.%s", ResourceType, Resource),
				ImportState:       true,
				ImportStateVerify: true,
			},
			// ... rest of the test
		},
	})
}
