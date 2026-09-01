# Enable a built-in framework for the whole organization.
# Destroy is state-only: it will NOT deselect this framework.
# Personal frameworks cannot hold the organization scope.
resource "orcasecurity_compliance_framework_selection" "gcp_cis" {
  framework_id = "gcp_cis_3.0.0"
  scopes       = ["organization"]
}

# Explicitly disable a framework. The plan shows the scope drop; apply DELETEs it.
resource "orcasecurity_compliance_framework_selection" "cost_optimization" {
  framework_id = "cost_optimization"
  scopes       = []
}
