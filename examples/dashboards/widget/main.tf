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

# Module outputs widgets_config — a list of { id, size } for built-in widgets only.
module "widget_builtins" {
  source = "./modules"
}

# Dashboard with built-in widgets only (from module)
resource "orcasecurity_custom_dashboard" "widget_test" {
  name               = "Widget ID Dashboard"
  organization_level = true
  filter_data        = {}
  view_type          = "dashboard"
  extra_params = {
    description    = "Dashboard with built-in widget IDs"
    version        = 2
    widgets_config = module.widget_builtins.widgets_config
  }
}

# Custom widgets to add to a dashboard
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

# Dashboard with both built-in (from module) and custom widget IDs
resource "orcasecurity_custom_dashboard" "mixed" {
  name               = "Widget ID Test Dashboard"
  organization_level = true
  filter_data        = {}
  view_type          = "dashboard"
  extra_params = {
    description = "Test built-in and custom widget IDs"
    version     = 2
    widgets_config = concat(
      module.widget_builtins.widgets_config,
      [
        { id = orcasecurity_custom_widget.example_1.id, size = "sm" },
        { id = orcasecurity_custom_widget.example_2.id, size = "sm" }
      ]
    )
  }
}

output "dashboard_id" {
  value = orcasecurity_custom_dashboard.widget_test.id
}

output "dashboard_mixed_id" {
  value = orcasecurity_custom_dashboard.mixed.id
}
