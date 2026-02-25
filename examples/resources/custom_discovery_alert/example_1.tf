# simple discovery-based custom alert
resource "orcasecurity_custom_discovery_alert" "alert_discovery" {
  name          = "Azure VNets that aren't in use"
  description   = "Azure VNets that don't have any compute or data resources attached to them via NICs."
  rule_json     = jsonencode({ "models" : ["AzureVNet"], "type" : "object_set", "with" : { "models" : ["AzureNetworkInterface"], "type" : "object_set", "keys" : ["NetworkInterfaces"], "operator" : "has", "negate" : true } })
  orca_score    = 6.2
  category      = "Network misconfigurations"
  context_score = false
}

# discovery-based custom alert with remediation steps
resource "orcasecurity_custom_discovery_alert" "example" {
  name          = "Azure VNets that aren't in use"
  description   = "Azure VNets that don't have any compute or data resources attached to them via NICs."
  rule_json     = jsonencode({ "models" : ["AzureVNet"], "type" : "object_set", "with" : { "models" : ["AzureNetworkInterface"], "type" : "object_set", "keys" : ["NetworkInterfaces"], "operator" : "has", "negate" : true } })
  orca_score    = 6.2
  category      = "Network misconfigurations"
  context_score = false
  remediation_text = {
    enable = true
    text   = "Add resources to this VNet via a network interface or delete this VNet."
  }
}

# discovery-based custom alert with custom compliance frameworks associations
resource "orcasecurity_custom_discovery_alert" "example" {
  name          = "Azure VNets that aren't in use"
  description   = "Azure VNets that don't have any compute or data resources attached to them via NICs."
  rule_json     = jsonencode({ "models" : ["AzureVNet"], "type" : "object_set", "with" : { "models" : ["AzureNetworkInterface"], "type" : "object_set", "keys" : ["NetworkInterfaces"], "operator" : "has", "negate" : true } })
  orca_score    = 6.2
  category      = "Network misconfigurations"
  context_score = false
  compliance_frameworks = [
    { name = "framework 1 name", section = "framework 1 section", priority = "low" },
    { name = "framework 2 name", section = "framework 2 section", priority = "medium" }
  ]
}
