terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

provider "orcasecurity" {
  api_endpoint = "https://api.orcasecurity.io"
  api_token = var.api_token
}

module "top10_widgets" {
  source = "./modules"
}

resource "orcasecurity_custom_dashboard" "security_category_dashboard" {
  name                    = "Top 10 Category Dashboard"
  organization_level      = true
  filter_data             = {}
  view_type               = "dashboard"
  extra_params = {
    description = "Top 10 alerts from all major alert categories"
    widgets_config = [
      {
        id = module.top10_widgets.malware_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.vulnerability_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.data_risk_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.iam_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.authentication_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.lateral_movement_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.suspicious_activity_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.malicious_activity_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.neglected_assets_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.network_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.authentication_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.ai_security_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.api_security_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.data_protection_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.file_integrity_alerts_widget_id
        size = "sm"
      },
      {
        id = module.top10_widgets.logging_monitoring_alerts_widget_id
        size = "sm"
      }
    ]
  }
}

