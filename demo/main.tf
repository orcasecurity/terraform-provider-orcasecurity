terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

provider "orcasecurity" {

}


data "orcasecurity_jira_template" "template_a" {
  template_name = "TF template"
}

output "jira_template_a" {
  value = data.orcasecurity_jira_template.template_a
}

resource "orcasecurity_automation" "jira_ticket" {
  name        = "Made with Terraform"
  description = "Automatically create JIRA issues for new alerts."
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "asset_regions", excludes : ["centralus"] },
    ]
  }

  jira_issue = {
    template_name = data.orcasecurity_jira_template.template_a.template_name
    parent_issue  = "1000500"
  }
}
