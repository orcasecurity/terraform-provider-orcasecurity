# PagerDuty integration
resource "orcasecurity_integration_pagerduty" "example" {
  template_name   = "pager_duty"
  integration_key = var.pagerduty_integration_key
  is_enabled      = true
  is_default      = false
}
