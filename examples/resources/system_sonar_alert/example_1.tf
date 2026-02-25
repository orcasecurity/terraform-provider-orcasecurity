# Disable a system alert
resource "orcasecurity_system_sonar_alert" "disable_api_gateway_alert" {
  rule_id = "r8ae477067a"
  enabled = false
}

# Enable a system alert
resource "orcasecurity_system_sonar_alert" "enable_alert" {
  rule_id = "r1234567890"
  enabled = true
}
