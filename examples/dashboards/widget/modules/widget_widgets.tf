# Widget example module: built-in widget IDs only.
# See docs/resources/custom_dashboard.md for the full widget ID reference.
terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

locals {
  widgets_config = [
    { id = "cloud-accounts-inventory", size = "sm" },
    { id = "security-score-benchmark", size = "md" },
    { id = "alerts-by-severity", size = "sm" },
    { id = "top-attack-path-entry-points", size = "sm" }
  ]
}

output "widgets_config" {
  value = local.widgets_config
}
