# ServiceNow credentials — backs both ITSM and SIR templates.
resource "orcasecurity_integration_servicenow_resource" "example" {
  name           = "my-servicenow"
  servicenow_url = "https://my-instance.service-now.com"
  username       = "username"
  password       = var.servicenow_password
}
