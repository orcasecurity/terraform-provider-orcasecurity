# Enable a built-in framework for the whole organization.
# Destroy is state-only by default: it will NOT deselect this framework.
resource "orcasecurity_compliance_framework_selection" "cis_aws" {
  framework_id = "cis_aws_foundations_1_4_0"
  scopes       = ["organization"]
}

# Explicitly disable a framework. The plan shows the scope drop; apply DELETEs it.
resource "orcasecurity_compliance_framework_selection" "cost_optimization" {
  framework_id = "cost_optimization"
  scopes       = []
}

# Opt destroy back in to restoring whatever scopes the framework had before Create.
resource "orcasecurity_compliance_framework_selection" "adopted" {
  framework_id       = "orca_best_practices"
  scopes             = ["user"]
  restore_on_destroy = true
}
