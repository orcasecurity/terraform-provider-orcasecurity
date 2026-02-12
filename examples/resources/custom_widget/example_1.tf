# table-type asset widget
resource "orcasecurity_custom_widget" "example_table" {
  name               = "GCP API Keys Table Widget"
  organization_level = true
  extra_params = {
    type                = "table"
    empty_state_message = "Widget query returned no data"
    default_size        = "sm"
    is_new              = true
    subtitle            = "API Keys Provisioned by GCP users"
    description         = "API Keys Provisioned by GCP users"
    settings = {
      columns = ["asset", "alertsOnAsset", "cloudAccount"]
      request_params = {
        query = jsonencode({
          models = ["GcpApiKey"]
          type   = "object_set"
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

# table-type alert widget
resource "orcasecurity_custom_widget" "example_alerts" {
  name               = "Alerts"
  organization_level = true
  extra_params = {
    type                = "alert-table"
    empty_state_message = "Widget query returned no data"
    default_size        = "sm"
    is_new              = true
    subtitle            = "Alerts"
    description         = "Alerts"
    settings = {
      columns = ["alert", "status", "priority"]
      request_params = {
        query = jsonencode({
          models = ["Alert"]
          type   = "object_set"
          with = {
            operator = "and"
            type     = "operation"
            values = [
              {
                key      = "Status"
                values   = ["open", "in_progress"]
                type     = "str"
                operator = "in"
              }
            ]
          }
        })
        group_by          = ["Name"]
        start_at_index    = 0
        order_by          = ["Score"]
        limit             = 10
        enable_pagination = true
      }
    }
  }
}

# donut-type widget (import: terraform import orcasecurity_custom_widget.example_donut <widget_id>)
resource "orcasecurity_custom_widget" "example_donut" {
  name               = "Inventory by Type"
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
