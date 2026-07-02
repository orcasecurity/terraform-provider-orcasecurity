# Jira Cloud template — opens issues in an Atlassian Cloud Jira project.
#
# The credentials side (the OAuth resource) must already exist in Orca. Grab its UUID from
# the Orca UI (Settings → Integrations → Jira Cloud) and paste it into `resource_id`.
resource "orcasecurity_integration_jira_cloud_template" "demo" {
  template_name         = "my-jira-cloud-template"
  resource_id           = "d24c8158-e466-4c6c-b40b-f3d86dd9a4fc"
  resource_url          = "https://my-tenant.atlassian.net"
  project_id            = "10000"
  issue_type_id         = "10001"
  subtask_issue_type_id = "10003"

  # Field mapping — keys are Jira field names; each value is a list of
  # `{ orca = "<alert_field>" }` (pull from the Orca alert) or
  # `{ value = "<literal>" }` (static). Multiple entries are concatenated when the Jira
  # field accepts a single value.
  mapping_json = jsonencode({
    summary = [
      { orca = "alert_name" },
      { orca = "alert_id" },
    ]
    description = [
      { orca = "details" },
      { orca = "alert_ui_link" },
      { orca = "recommendation" },
      { orca = "asset_details" },
      { orca = "account_name" },
      { orca = "findings" },
      { orca = "cloud_account_id" },
      { orca = "source" },
    ]
  })

  # Orca alert status → Jira workflow status ID
  alert_status_mapping_json = jsonencode({
    in_progress = "10001"
  })

  # Jira workflow status ID → Orca alert state change
  ticket_status_mapping_json = jsonencode({
    "10000" = {
      status      = "snoozed"
      snooze_days = 1
    }
  })

  # Same shape for sub-task tickets.
  subtask_alert_status_mapping_json = jsonencode({
    in_progress = "10001"
  })

  subtask_ticket_status_mapping_json = jsonencode({
    "10000" = {
      status = "closed"
    }
  })

  is_enabled = true
  is_default = false

  business_units = [
    "0b3a8907-1f43-44cb-9360-f18178ec1875",
    "a411f20b-0276-438c-a9d5-938c48a40957",
    "930c1e5e-6b2e-4881-a393-33491e758144",
  ]
}

output "jira_cloud_template_id" {
  value = orcasecurity_integration_jira_cloud_template.demo.id
}
