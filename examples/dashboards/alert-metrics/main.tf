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

module "alert_metrics_widgets" {
  source = "./modules"
}

resource "orcasecurity_custom_dashboard" "alert_metrics_dashboard" {
  name                    = "Alert Metrics"
  organization_level      = true
  filter_data             = {}
  view_type               = "dashboard"
  extra_params = {
    description = ""
    widgets_config = [
      {
        id = "alerts-by-severity"
        size = "sm"
      },
      {
        id = "risk-categories-alerts-over-time"
        size = "sm"
      },
      {
        id = "created-and-resolved-alerts-over-time"
        size = "sm"
      },
      {
        id = module.alert_metrics_widgets.closed_alerts_30_days_widget_id
        size = "sm"
      },
      {
        id = module.alert_metrics_widgets.closed_alerts_30_days_category_widget_id
        size = "sm"
      },
      {
        id = module.alert_metrics_widgets.closed_alerts_30_days_account_widget_id
        size = "sm"
      }
    ]
  }
}

