//custom widget resource
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
        subtitle = "Sample subtitle",
        description = "Sample description",
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
            }
        ]
    }
}