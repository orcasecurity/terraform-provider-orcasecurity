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

# Optional: Custom widget to add to a dashboard
resource "orcasecurity_custom_widget" "example" {
  name               = "Example Custom Widget"
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

# Dashboard mixing built-in (string IDs) and custom (resource ID)
resource "orcasecurity_custom_dashboard" "mixed" {
  name               = "Built-in and Custom Widgets Dashboard"
  organization_level = true
  filter_data        = {}
  view_type          = "dashboard"
  extra_params = {
    description = "Built-in + custom widgets"
    version     = 2
    widgets_config = concat(
      module.widget_builtins.widgets_config,
      [{ id = orcasecurity_custom_widget.example.id, size = "sm" }]
    )
  }
}

output "dashboard_id" {
  value = orcasecurity_custom_dashboard.widget_test.id
}
