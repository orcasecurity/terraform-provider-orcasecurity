package custom_widget_test

import (
	"regexp"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomWidgetResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_widget" "tf-custom-widget-1" {
  name               = "Custom Widget 1"
  organization_level = true

  extra_params = {
    type                = "donut"
    empty_state_message = "No data found"
    default_size        = "sm"
    is_new              = true
    subtitle            = "My Little Subtitle"
    description         = "My little description"
    settings = {
      field = {
        name = "Vm.Compute.Content.Inventory.Region"
        type = "str"
      }
      request_params = {
        query             = jsonencode({ models = ["AwsEc2Instance"], type = "object_set" })
        group_by          = ["Type"]
        group_by_list     = ["Vm.Compute.Content.Inventory.Region"]
        limit             = 0
        start_at_index    = 0
        enable_pagination = false
      }
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "name", "Custom Widget 1"),
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "extra_params.default_size", "sm"),
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "view_type", "customs_widgets"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_widget.tf-custom-widget-1",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_widget" "tf-custom-widget-1" {
  name               = "Custom Widget 2"
  organization_level = true

  extra_params = {
    type                = "donut"
    empty_state_message = "No data found"
    default_size        = "md"
    is_new              = false
    subtitle            = "My Little Subtitle"
    description         = "My little description"
    settings = {
      field = {
        name = "Vm.Compute.Content.Inventory.Region"
        type = "str"
      }
      request_params = {
        query             = jsonencode({ models = ["AwsEc2Instance"], type = "object_set" })
        group_by          = ["Type"]
        group_by_list     = ["Vm.Compute.Content.Inventory.Region"]
        limit             = 0
        start_at_index    = 0
        enable_pagination = false
      }
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "name", "Custom Widget 2"),
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "extra_params.default_size", "md"),
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "extra_params.subtitle", "My Little Subtitle"),
				),
			},
		},
	})
}

func TestAccCustomWidgetResource_UpdateWidgetQuery(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_widget" "tf-custom-widget-1" {
  name               = "Custom Widget 1"
  organization_level = true

  extra_params = {
    type                = "donut"
    empty_state_message = "No data found"
    default_size        = "sm"
    is_new              = true
    subtitle            = "Subtitle"
    description         = "Description"
    settings = {
      field = {
        name = "Vm.Compute.Content.Inventory.Region"
        type = "str"
      }
      request_params = {
        query             = jsonencode({ models = ["AwsS3Bucket"], type = "object_set" })
        group_by          = ["Type"]
        group_by_list     = ["Vm.Compute.Content.Inventory.Region"]
        limit             = 0
        start_at_index    = 0
        enable_pagination = false
      }
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "extra_params.settings.request_params.query", regexp.MustCompile("AwsS3Bucket")),
				),
			},
			// update - change query model
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_widget" "tf-custom-widget-1" {
  name               = "Custom Widget 1"
  organization_level = true

  extra_params = {
    type                = "donut"
    empty_state_message = "No data found"
    default_size        = "sm"
    is_new              = false
    subtitle            = "Subtitle"
    description         = "Description"
    settings = {
      field = {
        name = "Vm.Compute.Content.Inventory.Region"
        type = "str"
      }
      request_params = {
        query             = jsonencode({ models = ["AwsEc2Instance"], type = "object_set" })
        group_by          = ["Type"]
        group_by_list     = ["Vm.Compute.Content.Inventory.Region"]
        limit             = 0
        start_at_index    = 0
        enable_pagination = false
      }
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "extra_params.settings.request_params.query", regexp.MustCompile("AwsEc2Instance")),
				),
			},
		},
	})
}
