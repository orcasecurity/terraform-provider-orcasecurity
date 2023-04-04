// basic usage
resource "orcasecurity_automation" "example" {
  name        = "JIRA issues"
  description = "Automatically create JIRA issues"
  jira_issue = {
    template_name = "My Template"
  }
}

// cretate issue in parent ticket
resource "orcasecurity_automation" "example" {
  name        = "JIRA issues"
  description = "Automatically create JIRA issue in parent issue"
  jira_issue = {
    template_name = "My Template"
    parent_issue  = "ABC-007"
  }
}
