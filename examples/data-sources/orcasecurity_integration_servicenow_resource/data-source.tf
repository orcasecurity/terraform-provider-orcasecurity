# Look up an existing ServiceNow credentials resource by name and feed its id into a
# template — useful when the credentials were created via the UI or a different Terraform
# workspace. The same credentials resource backs both ITSM and SIR templates.
data "orcasecurity_integration_servicenow_resource" "existing" {
  name = "servicenow-prod"
}

resource "orcasecurity_integration_servicenow_itsm_template" "demo" {
  template_name = "Yuri Demo"
  resource_id   = data.orcasecurity_integration_servicenow_resource.existing.id
  instance_name = "ven03666"

  resolution_status = "6"
  resolution_code   = "Resolved by caller"
  resolution_note   = "Resolved by Yuri"

  mapping_json = jsonencode({
    category    = [{ value = "software" }]
    u_risk_name = [{ orca = "alert_name" }]
  })
}
