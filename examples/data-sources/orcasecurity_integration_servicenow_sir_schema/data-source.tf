# Read the list of fields a ServiceNow SIR resource exposes. Useful for authoring or
# auditing the `mapping_json` argument on `orcasecurity_integration_servicenow_sir_template`.
data "orcasecurity_integration_servicenow_sir_schema" "fields" {
  resource_id = "a6188626-fd15-4c0d-a262-9f3deffb5459"
}

output "sir_elements" {
  value = data.orcasecurity_integration_servicenow_sir_schema.fields.elements
}

output "sir_fields" {
  value = data.orcasecurity_integration_servicenow_sir_schema.fields.fields
}
