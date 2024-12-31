package custom_dashboard_test

import (
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	ResourceType = "orcasecurity_custom_dashboard"
	Resource     = "terraformTestResource"
	OrcaObject   = "terraformTestResourceInOrca"
)

func TestAccCustomDashboardResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
    name = "%s"
    filter_data = {}
    organization_level = true
    view_type = "dashboard"
    extra_params = {
        description = "my 1st simple dashboard"
        widgets_config = [
            {
                id = "cloud-accounts-inventory"
                size = "sm"
            },
            {
                id = "security-score-benchmark"
                size = "md"
            }
        ]
    }
            }`, ResourceType, Resource, OrcaObject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "name", fmt.Sprintf("%s", OrcaObject)),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "organizational_level", "true"),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "view_type", "dashboard"),
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
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_dashboard" "tf-custom-dash-1" {
    name = "Orca Custom Dashboard 2"
    filter_data = {}
    organization_level = false
    view_type = "dashboard"
    extra_params = {
        description = "my 2nd simple dashboard"
        widgets_config = [
            {
                id = "attack-paths"
                size = "sm"
            },
            {
                id = "trending-news"
                size = "md"
            }
        ]
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "name", "Orca Custom Dashboard 2"),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "organizational_level", "false"),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "view_type", "dashboard"),
				),
			},
		},
	})
}

func TestAccCustomDashboardResource_UpdateWidgets(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_dashboard" "tf-custom-dash-1" {
    name = "Orca Custom Dashboard 1"
    filter_data = {}
    organization_level = true
    view_type = "dashboard"
    extra_params = {
        description = "my 1st simple dashboard"
        widgets_config = [
            {
                id = "cloud-accounts-inventory"
                size = "sm"
            },
            {
                id = "security-score-benchmark"
                size = "md"
            }
        ]
    }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "extra_params.widgets_config[0].id", "cloud-accounts-inventory"),
				),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_dashboard" "tf-custom-dash-1" {
    name = "Orca Custom Dashboard 1"
    filter_data = {}
    organization_level = true
    view_type = "dashboard"
    extra_params = {
        description = "my 1st simple dashboard"
        widgets_config = [
            {
                id = "attack-paths"
                size = "sm"
            },
            {
                id = "trending-news"
                size = "md"
            }
        ]
    }
}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "extra_params.widgets_config[0].id", "attack-paths"),
				),
			},
		},
	})
}
