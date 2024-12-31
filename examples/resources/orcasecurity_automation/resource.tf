# automation that sends alerts to Jira
resource "orcasecurity_automation" "example" {
  name        = "Jira"
  description = "Send high, critical alerts on our central resources to the SecOps Jira template (project)"
  enabled     = true
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "asset_regions", includes : ["centralus"] },
    ]
  }
  jira_issue = {
    template_name = "SecOps"  # name of Jira template
    parent_issue  = "JMB-007" // optional
  }
}

# automation that sends alerts to Sumo Logic
resource "orcasecurity_automation" "example" {
  name        = "Sumo Logic"
  description = "Send high, critical alerts on our central resources to our Sumo Logic instance"
  enabled     = true
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "asset_regions", includes : ["centralus"] },
    ]
  }
  sumologic = {

  }
}

# automation that sends alert data to a Webhook URL
resource "orcasecurity_automation" "example" {
  name        = "Webhook"
  description = "Send high, critical alerts on our central resources to a Webhook URL"
  enabled     = true
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "asset_regions", includes : ["centralus"] },
    ]
  }
  webhook = {
    name = "my-webhook-name" # name of Webhook template
  }
}

# automation that sends alert data to email addresses
resource "orcasecurity_automation" "example" {
  name        = "Email"
  description = "Send high, critical alerts on our central resources to the 2 Does"
  enabled     = true
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "asset_regions", includes : ["centralus"] },
    ]
  }
  email_template = {
    email        = ["john.doe@orca.security", "jane.doe@orca.security"]
    multi_alerts = true
  }
}

# automation that dismisses alerts on a specific business unit
resource "orcasecurity_automation" "example" {
  name           = "Dismiss alerts"
  description    = "Dismiss high, critical alerts on our central resources"
  business_units = ["43008048-8a3f-4daa-8e38-290842d28c62"]
  enabled        = true
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "asset_regions", includes : ["centralus"] },
    ]
  }

  alert_dismissal_details = {
    reason        = "Non-Production"
    justification = "These are test applications, not production applications."
  }
}

# automation that changes the risk score of alerts
resource "orcasecurity_automation" "example" {
  name        = "Change Orca Score to 3.0"
  description = "Decrease alert scores of high, critical alerts on our central resources"
  enabled     = true
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "asset_regions", includes : ["centralus"] },
    ]
  }

  alert_score_specify_details = {
    new_score     = 2.7
    reason        = "Non-Production"
    justification = "These are test applications, not production applications."
  }
}

# automation that decreases the risk score of alerts
resource "orcasecurity_automation" "example" {
  name        = "Decrease score"
  description = "Decrease alert scores of high, critical alerts on our central resources"
  enabled     = true
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "asset_regions", includes : ["centralus"] },
    ]
  }

  alert_score_decrease_details = {
    reason        = "Non-Production"
    justification = "These are test applications, not production applications."
  }
}

# automation that increases the risk score of alerts
resource "orcasecurity_automation" "example" {
  name        = "Increase score"
  description = "Increase alert scores of medium, high alerts on our central resources"
  enabled     = true
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["medium", "high"] },
      { field : "asset_regions", excludes : ["centralus"] },
    ]
  }

  alert_score_increase_details = {
    reason        = "Non-Production"
    justification = "These are test applications, not production applications."
  }
}

# automation that sends alerts to a Slack channel
resource "orcasecurity_automation" "example" {
  name        = "Slack"
  description = "Send high, critical malware alerts on our central resources to the SecOps Slack channel"
  enabled     = true
  query = {
    filter : [
      { field : "state.status", includes : ["open"] },
      { field : "state.risk_level", includes : ["high", "critical"] },
      { field : "category", includes : ["malware"] },
      { field : "asset_regions", includes : ["centralus"] },
    ]
  }

  slack_template = {
    workspace = "My Company Workspace"
    channel   = "C04CLAAAAA"
  }
}
