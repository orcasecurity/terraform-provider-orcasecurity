terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

provider "orcasecurity" {
  api_endpoint = var.api_endpoint
  api_token    = var.api_token
}

# Custom widgets to include on the dashboard
resource "orcasecurity_custom_widget" "example_1" {
  name               = "Example Custom Widget 1"
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
        query         = jsonencode({ models = ["Inventory"], type = "object_set" })
        group_by      = ["Type"]
        group_by_list = ["CloudAccount.Name"]
      }
      field = { name = "Type", type = "str" }
    }
  }
}

resource "orcasecurity_custom_widget" "example_2" {
  name               = "Example Custom Widget 2"
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
        query         = jsonencode({ models = ["Inventory"], type = "object_set" })
        group_by      = ["Type"]
        group_by_list = ["CloudAccount.Name"]
      }
      field = { name = "Type", type = "str" }
    }
  }
}

# Dashboard with inline widgets_config: built-in widget IDs (strings) and custom widget IDs (resource reference)
resource "orcasecurity_custom_dashboard" "widget_test" {
  name               = "Widget ID Test Dashboard"
  organization_level = true
  filter_data        = {}
  view_type          = "dashboard"
  extra_params = {
    description = "Test built-in and custom widget IDs"
    version     = 2
    widgets_config = [
      { id = "cloud-accounts-inventory", size = "sm" },
      { id = "security-score-benchmark", size = "md" },
      { id = "alerts-by-severity", size = "sm" },
      { id = orcasecurity_custom_widget.example_1.id, size = "sm" },
      { id = orcasecurity_custom_widget.example_2.id, size = "sm" }
    ]
  }
}

output "dashboard_id" {
  value = orcasecurity_custom_dashboard.widget_test.id
}
