terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

provider "orcasecurity" {

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
    template_name = "Template A"
    parent_issue = "1000500"
  }
}
