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

module "vuln_widgets" {
  source = "./modules"
}

resource "orcasecurity_custom_dashboard" "vulnerabilities_dashboard" {
  name               = "Vulnerability Management"
  organization_level = true
  filter_data        = {}
  view_type          = "dashboard"
  extra_params = {
    description = ""
    widgets_config = [
      {
        id   = module.vuln_widgets.vulnerability_alerts_widget_id
        size = "sm"
      },
      {
        id   = "fixable-vulnerabilities"
        size = "sm"
      },
      {
        id   = "assets-max-cvss"
        size = "sm"
      },
      {
        id   = module.vuln_widgets.priority_vulnerabilities_alerts_widget_id
        size = "sm"
      },
      {
        id   = module.vuln_widgets.cisa_kev_public_assets_widget_id
        size = "sm"
      },
      {
        id   = module.vuln_widgets.fixable_critical_vulnerabilities_widget_id
        size = "sm"
      },
      {
        id   = module.vuln_widgets.priority_vulnerabilities_alerts_source_widget_id
        size = "sm"
      },
      {
        id   = module.vuln_widgets.prevalent_critical_vulnerabilities_widget_id
        size = "sm"
      }
    ]
  }
}

