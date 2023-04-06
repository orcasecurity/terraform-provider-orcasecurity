// usage with jira template
resource "orcasecurity_automation" "example" {
  name        = "JIRA issues"
  description = "Automatically create JIRA issues"
  jira_issue = {
    template_name = "My Template"
    parent_issue  = "ABC-007" // optional
  }
}
