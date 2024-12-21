//custom discovery-based alert
resource "orcasecurity_custom_discovery_alert" "alert_discovery" {
  name          = "custom-alert"
  rule_json     = jsonencode({ "models" : ["AzureAksCluster"], "type" : "object_set" })
  category      = "Lateral movement"
  orca_score    = 5.0
  description   = "description"
  severity      = 1
  context_score = true
  compliance_frameworks = [
    {
      name     = "test-fw"
      section  = "Account"
      priority = "medium"
    }
  ]
}