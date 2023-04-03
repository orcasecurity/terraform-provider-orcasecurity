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
  name        = "Create JIRA ticket"
  description = "Automatically create JIRA issues for new alerts."
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] }
    ]
  }
}
