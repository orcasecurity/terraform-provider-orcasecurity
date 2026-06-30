# Credentials side of the integration — created separately.
resource "orcasecurity_integration_servicenow_resource" "creds" {
  name           = "servicenow-itsm-prod"
  servicenow_url = "https://ven03666.service-now.com"
  username       = "username"
  password       = var.servicenow_password
}

# Template that defines how Orca alerts map to ServiceNow incident fields and how
# resolution/reopen events behave.
resource "orcasecurity_integration_servicenow_itsm_template" "demo" {
  template_name = "teamplate_name"
  resource_id   = orcasecurity_integration_servicenow_resource.creds.id
  instance_name = "instance_name"
  username      = "username"

  resolution_status = "6"
  resolution_code   = "Resolved by caller"
  resolution_note   = "Resolved by @someone"
  reopen_status     = "7"

  mapping_json = jsonencode({
    category        = [{ value = "software" }]
    u_risk_name     = [{ orca = "alert_name" }]
    u_subscription  = []
    business_impact = [{ orca = "details" }]
  })

  is_enabled = true
  is_default = false
}
