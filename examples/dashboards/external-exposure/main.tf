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

module "external_exposure_widgets" {
  source = "./modules"
}

resource "orcasecurity_custom_dashboard" "external_exposure_dashboard" {
  name                    = "External Exposure"
  organization_level      = true
  filter_data             = {}
  view_type               = "dashboard"
  extra_params = {
    description = ""
    widgets_config = [
      {
        id = "alerts-on-exposed-assets"
        size = "sm"
      },
      {
        id = module.external_exposure_widgets.public_facing_assets_widget_id
        size = "sm"
      },
      {
        id = "top-attack-path-entry-points"
        size = "sm"
      },
      {
        id = module.external_exposure_widgets.vm_public_ingress_widget_id
        size = "sm"
      },
      {
        id = module.external_exposure_widgets.public_facing_neglected_compute_widget_id
        size = "sm"
      },
      {
        id = module.external_exposure_widgets.attack_paths_public_widget_id
        size = "sm"
      },
      {
        id = module.external_exposure_widgets.public_assets_priority_risk_widget_id
        size = "sm"
      },
      {
        id = module.external_exposure_widgets.cisa_kev_public_assets_widget_id
        size = "sm"
      },
      {
        id = module.external_exposure_widgets.public_data_widget_id
        size = "sm"
      }
    ]
  }
}

