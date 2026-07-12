# Monday.com credentials resource, created from a Monday API token. Orca derives the
# account slug from the token and stores the token in its secret store. Use the returned
# id as resource_id on orcasecurity_integration_monday_template.
resource "orcasecurity_integration_monday_resource" "demo" {
  name      = "monday-prod"
  api_token = var.monday_api_token
}

output "monday_resource_id" {
  value = orcasecurity_integration_monday_resource.demo.id
}
