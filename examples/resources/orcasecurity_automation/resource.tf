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
    template_name = "My Template" // as on Orca dashboard
    parent_issue  = "JMB-007"     // optional
  }
}

// usage with Sumo Logic integration
resource "orcasecurity_automation" "example" {
  name        = "Sumo Logic"
  description = "Integrate Sumo Logic"
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

// usage with web hooks
resource "orcasecurity_automation" "example" {
  name        = "Automation with web hook"
  description = "Automatically submit data to web hook"
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "asset_regions", excludes : ["centralus"] },
    ]
  }
  webhook = {
    name = "my-webhook-name" // as on Orca dashboard
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
