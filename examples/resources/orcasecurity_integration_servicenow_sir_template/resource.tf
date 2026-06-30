# Credentials side of the integration — created separately.
resource "orcasecurity_integration_servicenow" "creds" {
  name           = "cred_name"
  servicenow_url = "https://instance_name.service-now.com"
  username       = "username"
  password       = var.servicenow_password
}

# Optional — discover the available SIR fields so you know what's valid in mapping_json.
data "orcasecurity_integration_servicenow_sir_schema" "fields" {
  resource_id = orcasecurity_integration_servicenow.creds.id
}

output "sir_field_names" {
  value = data.orcasecurity_integration_servicenow_sir_schema.fields.elements
}

resource "orcasecurity_integration_servicenow_sir_template" "demo" {
  template_name = "teamplate_name"
  resource_id   = orcasecurity_integration_servicenow.creds.id

  resolution_status = "10"
  resolution_code   = "-100"
  resolution_note   = "resolution_note"
  reopen_status     = "10"

  mapping_json = jsonencode({
    affected_user            = [{ value = "Affected user" }]
    change_request           = [{ value = "Change request" }]
    malware_url              = [{ value = "Malware URL" }]
    problem                  = [{ value = "Problem" }]
    alert_rule               = [{ value = "Alert Rule" }]
    risk_score               = [{ value = "5" }]
    business_criticality     = [{ value = "1" }]
    other_ioc                = [{ value = "Other IoC" }]
    parent_security_incident = [{ value = "Parent security incident" }]
    incident                 = [{ value = "Incident" }]
    source_ip                = [{ value = "Source IP" }]
    risk                     = [{ value = "4" }]
    vulnerability            = [{ value = "Vulnerability" }]
    dest_ip                  = [{ value = "Destination IP" }]
    risk_change              = [{ value = "down" }]
    alert_sensor             = [{ value = "Alert Sensor" }]
    integration_source       = [{ value = "Integration Source" }]
    severity                 = [{ value = "1" }]
    malware_hash             = [{ value = "Malware hash" }]
    referrer_url             = [{ value = "Referrer URL" }]
    assigned_vendor          = [{ value = "Assigned vendor" }]
    template                 = [{ value = "Template" }]
    opened_for               = [{ value = "Opened for" }]
    sla_suspended_for        = [{ value = "Suspended for" }]
    initiated_from           = [{ value = "Initiated from" }]
    contact_type             = [{ value = "endpoint_security" }]
  })

  on_close_alert_mapping_json = jsonencode({})

  is_enabled = true
  is_default = false
}
