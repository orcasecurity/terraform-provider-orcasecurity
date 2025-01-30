terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

provider "orcasecurity" {
  api_endpoint = "https://api.orcasecurity.io"
  api_token    = var.api_token
}

module "secops_widgets" {
  source = "./modules"
}

resource "orcasecurity_custom_dashboard" "secops_dashboard" {
  name               = "SecOps"
  organization_level = true
  filter_data        = {}
  view_type          = "dashboard"
  extra_params = {
    description = ""
    widgets_config = [
      {
        id   = "alerts-on-exposed-assets"
        size = "sm"
      },
      {
        id   = module.secops_widgets.malware_event_monitoring_widget_id
        size = "sm"
      },
      {
        id   = module.secops_widgets.real_time_sensor_widget_id
        size = "sm"
      },
      {
        id   = module.secops_widgets.cspm_secure_config_widget_id
        size = "sm"
      },
      {
        id   = module.secops_widgets.top_cspm_alerts_widget_id
        size = "sm"
      },
      {
        id   = module.secops_widgets.top_sensor_alerts_widget_id
        size = "sm"
      },
      {
        id   = "top-attack-path-entry-points"
        size = "sm"
      },
      {
        id   = "alerts-pending-action"
        size = "md"
      }
    ]
  }
}

