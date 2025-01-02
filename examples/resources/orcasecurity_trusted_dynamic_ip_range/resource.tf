# Simple with data source
data "orcasecurity_organization" "myorganization" {}

output "myorganization" {
  value = data.orcasecurity_organization.myorganization.name
}

resource "orcasecurity_dynamic_trusted_ip_range" "example" {
  org_id  = output.myorganization.value
  enabled = true
}