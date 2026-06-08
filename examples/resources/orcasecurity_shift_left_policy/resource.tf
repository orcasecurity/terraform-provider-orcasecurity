resource "orcasecurity_shift_left_policy" "iac_baseline" {
  type                       = "iac"
  name                       = "IaC baseline"
  description                = "Managed by Terraform"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"

  iac {
    controls {
      title    = "API Gateway is publicly accessible"
      priority = "HIGH"
      disabled = false
    }
  }
}
