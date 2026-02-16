# table-type asset widget
resource "orcasecurity_custom_widget" "example_table" {
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
            "GcpVmInstance",
            "Vm",
            "VmwareVmInstance",
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

# donut-type widget (import: terraform import orcasecurity_custom_widget.example_donut <widget_id>)
resource "orcasecurity_custom_widget" "example_donut" {
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
