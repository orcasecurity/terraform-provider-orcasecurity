# Read the list of fields a ServiceNow resource exposes for a given table variant. Useful for
# authoring or auditing the `mapping_json` argument on the SIR/ITSM template resources.
# SIR and ITSM share the same credentials resource but map to different tables, so set `type`
# to the variant you are configuring.

# Security Incident Response (sn_si_incident table)
data "orcasecurity_integration_servicenow_schema" "sir" {
  resource_id = "a6188626-fd15-4c0d-a262-9f3deffb5459"
  type        = "sir"
}

# IT Service Management (incident table)
data "orcasecurity_integration_servicenow_schema" "itsm" {
  resource_id = "a6188626-fd15-4c0d-a262-9f3deffb5459"
  type        = "itsm"
}

output "sir_elements" {
  value = data.orcasecurity_integration_servicenow_schema.sir.elements
}

output "itsm_elements" {
  value = data.orcasecurity_integration_servicenow_schema.itsm.elements
}
