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

module "widget_builtins" {
  source = "./modules"
}

resource "orcasecurity_custom_dashboard" "widget_test" {
  name               = "Widget ID Dashboard"
  organization_level = true
  filter_data        = {}
  view_type          = "dashboard"
  extra_params = {
    description    = "Dashboard with built-in widget IDs"
    widgets_config = module.widget_builtins.widgets_config
  }
}

output "dashboard_id" {
  value = orcasecurity_custom_dashboard.widget_test.id
}
