# Slack integration that posts Orca alerts to a Slack workspace/channel. The mapping
# lists which Orca alert fields render in each message section (title, description).
resource "orcasecurity_integration_slack_template" "demo" {
  template_name = "tf_slack"
  workspace_id  = "T0A0KSCQ1B3"
  channels      = ["C0AE82CGDH7"]
  show_actions  = true

  mapping = {
    title = [
      "alert_id", "orca_score", "source", "alert_labels",
      "type", "asset_type", "status", "risk_level",
    ]
    description = [
      "details", "cloud_provider", "account_name", "asset_name",
      "asset_state", "recommendation", "cve_list", "findings",
    ]
  }

  business_units = [
    "a411f20b-0276-438c-a9d5-938c48a40957",
    "930c1e5e-6b2e-4881-a393-33491e758144",
  ]

  is_enabled = true
  is_default = false
}
