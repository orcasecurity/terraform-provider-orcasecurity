# Look up an existing Monday.com credentials resource by name (e.g. created in the Orca UI
# or a different workspace) and reuse its id as resource_id on a Monday template.
data "orcasecurity_integration_monday_resource" "existing" {
  name = "monday-prod"
}

resource "orcasecurity_integration_monday_template" "demo" {
  template_name = "monday-template-name"
  resource_id   = data.orcasecurity_integration_monday_resource.existing.id
  workspace_id  = "3709069"
  board_id      = "1827821929"

  mapping_json = jsonencode({
    text = ["alert_id", "asset_name"]
  })
}
