// usage with jira template
resource "orcasecurity_automation" "example" {
  name        = "JIRA issues"
  description = "Automatically create JIRA issues"
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "asset_regions", excludes : ["centralus"] },
    ]
  }
  jira_issue = {
    template_name = "My Template"
    parent_issue  = "JMB-007" // optional
  }
}

// usage witn Sumo Logic integration
resource "orcasecurity_automation" "example" {
  name        = "JIRA issues"
  description = "Automatically create JIRA issues"
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "asset_regions", excludes : ["centralus"] },
    ]
  }
  sumologic = {

  }
}
