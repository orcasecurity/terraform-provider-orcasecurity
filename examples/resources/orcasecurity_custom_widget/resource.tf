# table-type widget
resource "orcasecurity_custom_widget" "example" {
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
          "models" : [
            "GcpApiKey"
          ],
          "type" : "object_set"
        })
        group_by : [
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

# donut-type widget
resource "orcasecurity_custom_widget" "example-2" {
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
          "models" : [
            "GcpApiKey"
          ],
          "type" : "object_set"
        })
        group_by : [
          "Type"
        ],
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