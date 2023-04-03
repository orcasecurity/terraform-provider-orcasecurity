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
}
