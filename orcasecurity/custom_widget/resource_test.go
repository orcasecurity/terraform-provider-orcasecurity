package custom_widget_test

import (
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
    name = "Custom Widget 1"

    organization_level = true
    view_type = "customs_widgets"
    filter_data = {}
    

    extra_params = {
        type = "PIE_CHART_SINGLE",
        category = "Custom",
        empty_state_message = "No data found",
        size = "sm",
        is_new = true,
        title = "Custom Widget 1",
        subtitle = "My Little Subtitle",
        description = "My little description",
        settings = [
            {
                size = "sm",
                field = {
                    name = "Vm.Compute.Content.Inventory.Region",
                    type = "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsEc2Instance"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            },
            {
                size = "md",
                field = {
                    name = "Vm.Compute.Content.Inventory.Region",
                    type = "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsEc2Instance"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            },
            {
                size = "lg",
                field = {
                    "name": "Vm.Compute.Content.Inventory.Region",
                    "type": "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsEc2Instance"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            }]
            }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "name", "Custom Widget 1"),
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "extra_params.size", "sm"),
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "view_type", "dashboard"),
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
    name = "Custom Widget 2"

    organization_level = true
    view_type = "customs_widgets"
    filter_data = {}
    

    extra_params = {
        type = "PIE_CHART_SINGLE",
        category = "Custom",
        empty_state_message = "No data found",
        size = "md",
        is_new = true,
        title = "Custom Widget 1",
        subtitle = "My Little Subtitle",
        description = "My little description",
        settings = [
            {
                size = "sm",
                field = {
                    name = "Vm.Compute.Content.Inventory.Region",
                    type = "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsEc2Instance"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            },
            {
                size = "md",
                field = {
                    name = "Vm.Compute.Content.Inventory.Region",
                    type = "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsEc2Instance"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            },
            {
                size = "lg",
                field = {
                    "name": "Vm.Compute.Content.Inventory.Region",
                    "type": "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsEc2Instance"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            }]
            }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "name", "Custom Widget 2"),
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "extra_params.size", "md"),
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
    name = "Custom Widget 1"

    organization_level = true
    view_type = "customs_widgets"
    filter_data = {}
    

    extra_params = {
        type = "PIE_CHART_SINGLE",
        category = "Custom",
        empty_state_message = "No data found",
        size = "sm",
        is_new = true,
        title = "Custom Widget 1",
        subtitle = "My Little Subtitle",
        description = "My little description",
        settings = [
            {
                size = "sm",
                field = {
                    name = "Vm.Compute.Content.Inventory.Region",
                    type = "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsS3Bucket"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            },
            {
                size = "md",
                field = {
                    name = "Vm.Compute.Content.Inventory.Region",
                    type = "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsS3Bucket"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            },
            {
                size = "lg",
                field = {
                    "name": "Vm.Compute.Content.Inventory.Region",
                    "type": "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsS3Bucket"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            }]
            }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "extra_params.settings[0].request_params", `jsonencode({\"query\": {
                        \"models\": [
                            \"AwsS3Bucket\"
                        ],
                        \"type\": \"object_set\"
                    },
                    \"additional_models[]\": [
                        \"CloudAccount\",
                        \"CodeOrigins\",
                        \"CustomTags\"
                    ],
                    \"group_by\": [
                        \"Type\"
                    ],
                    \"group_by[]\": [
                        \"Vm.Compute.Content.Inventory.Region\"
                    ]
                })`),
				),
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_widget" "tf-custom-widget-1" {
    name = "Custom Widget 1"

    organization_level = true
    view_type = "customs_widgets"
    filter_data = {}
    

    extra_params = {
        type = "PIE_CHART_SINGLE",
        category = "Custom",
        empty_state_message = "No data found",
        size = "sm",
        is_new = true,
        title = "Custom Widget 1",
        subtitle = "My Little Subtitle",
        description = "My little description",
        settings = [
            {
                size = "sm",
                field = {
                    name = "Vm.Compute.Content.Inventory.Region",
                    type = "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsEc2Instance"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            },
            {
                size = "md",
                field = {
                    name = "Vm.Compute.Content.Inventory.Region",
                    type = "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsEc2Instance"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            },
            {
                size = "lg",
                field = {
                    "name": "Vm.Compute.Content.Inventory.Region",
                    "type": "str"
                },
                request_params = jsonencode({
                    "query": {
                        "models": [
                            "AwsEc2Instance"
                        ],
                        "type": "object_set"
                    },
                    "additional_models[]": [
                        "CloudAccount",
                        "CodeOrigins",
                        "CustomTags"
                    ],
                    "group_by": [
                        "Type"
                    ],
                    "group_by[]": [
                        "Vm.Compute.Content.Inventory.Region"
                    ]
                })
            }]
            }
}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_widget.tf-custom-widget-1", "extra_params.settings[0].request_params", `jsonencode({\"query\": {
                        \"models\": [
                            \"AwsEc2Instance\"
                        ],
                        \"type\": \"object_set\"
                    },
                    \"additional_models[]\": [
                        \"CloudAccount\",
                        \"CodeOrigins\",
                        \"CustomTags\"
                    ],
                    \"group_by\": [
                        \"Type\"
                    ],
                    \"group_by[]\": [
                        \"Vm.Compute.Content.Inventory.Region\"
                    ]
                })`),
				),
			},
		},
	})
}
