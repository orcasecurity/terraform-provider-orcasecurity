# table-type asset widget (Serving Layer (SVL) Orca backend)
resource "orcasecurity_custom_widget" "example_1" {
  name               = "Virtual Instances"
  organization_level = true
  extra_params = {
    type                = "table"
    empty_state_message = "Widget query returned no data"
    default_size        = "sm"
    is_new              = true
    subtitle            = "Virtual Instances"
    description         = "Virtual Instances"
    settings = {
      columns = ["asset", "alertsOnAsset", "cloudAccount"]
      request_params = {
        query = jsonencode({
          models = [
            "AliCloudEcsInstance",
            "AwsEc2Instance",
            "AzureComputeVm",
            "AzureComputeVmAvailabilitySet",
            "AzureComputeVmScaleSets",
            "GcpComputeInstanceGroup",
            "GcpComputeTargetPool",
            "GcpVmInstance",
            "OciComputeVmInstance",
            "OnPremVm",
            "Vm",
            "VmwareVmInstance",
            "TencentCloudCvmInstance",
            "ByocVm",
            "LinodeInstance"
          ]
          type = "object_set"
        })
        group_by          = ["Type"]
        start_at_index    = 0
        order_by          = ["-Inventory.OrcaScore"]
        limit             = 10
        enable_pagination = true
      }
    }
  }
}

# donut-type widget - import: terraform import orcasecurity_custom_widget.test_widget <widget_id>
resource "orcasecurity_custom_widget" "test_widget" {
  name               = "Test Custom Widget"
  organization_level = true
  extra_params = {
    type                = "donut"
    empty_state_message = "No data found"
    default_size        = "sm"
    is_new              = true
    subtitle            = ""
    description         = ""
    settings = {
      request_params = {
        query = jsonencode({
          models = ["Inventory"]
          type   = "object_set"
        })
        group_by      = ["Type"]
        group_by_list = ["CloudAccount.Name"]
      }
      field = {
        name = "Type"
        type = "str"
      }
    }
  }
}
