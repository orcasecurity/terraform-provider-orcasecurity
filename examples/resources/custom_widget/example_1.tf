# table-type asset widget (Serving Layer (SVL) Orca backend)
resource "orcasecurity_custom_widget" "example_1" {
  name               = "GCP API Keys Table Widget"
  organization_level = true
  extra_params = {
    type                = "table",
    empty_state_message = "Widget query returned no data",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "API Keys Provisioned by GCP users",
    description         = "API Keys Provisioned by GCP users",
    settings = {
      columns = [
        "asset",
        "alertsOnAsset",
        "cloudAccount"
      ]
      request_params = {
        query = jsonencode({
          models = ["GcpApiKey"]
          type   = "object_set"
        })
        group_by = [
          "Type"
        ]
        start_at_index    = 0
        order_by          = ["-Inventory.OrcaScore"]
        limit             = 10
        enable_pagination = true
      }
    }
  }
}

# table-type asset widget (legacy Orca backend)
resource "orcasecurity_custom_widget" "example_2" {
  name               = "GCP API Keys Table Widget 2"
  organization_level = true
  extra_params = {
    type                = "asset-table",
    empty_state_message = "Widget query returned no data",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "API Keys Provisioned by GCP users",
    description         = "API Keys Provisioned by GCP users",
    settings = {
      columns = [
        "asset",
        "alertsOnAsset",
        "cloudAccount"
      ]
      request_params = {
        query = jsonencode({
          models = ["GcpApiKey"]
          type   = "object_set"
        })
        group_by = [
          "Type"
        ]
        start_at_index    = 0
        order_by          = ["-Inventory.OrcaScore"]
        limit             = 10
        enable_pagination = true
      }
    }
  }
}


# table-type alert widget (legacy Orca backend)
resource "orcasecurity_custom_widget" "example_3" {
  name               = "Alerts"
  organization_level = true
  extra_params = {
    type                = "alert-table",
    empty_state_message = "Widget query returned no data",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "Alerts",
    description         = "Alerts",
    settings = {
      columns = [
        "alert",
        "status",
        "priority"
      ]
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
        group_by = [
          "Name"
        ]
        start_at_index    = 0
        order_by          = ["Score"]
        limit             = 10
        enable_pagination = true
      }
    }
  }
}

# donut-type widget with Inventory models (import: terraform import orcasecurity_custom_widget.test_widget <widget_id>)
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

# donut-type widget
resource "orcasecurity_custom_widget" "example_4" {
  name               = "GCP API Keys Donut Widget"
  organization_level = true
  extra_params = {
    type                = "donut",
    empty_state_message = "Widget query returned no data",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "API Keys Provisioned by GCP users",
    description         = "API Keys Provisioned by GCP users",
    settings = {
      request_params = {
        query = jsonencode({
          models = ["GcpApiKey"]
          type   = "object_set"
        })
        group_by = ["Type"]
        group_by_list = [
          "CloudAccount.Name"
        ]
      }
      field = {
        name = "CloudAccount.Name",
        type = "str"
      }
    }
  }
}
