data "orcasecurity_organization" "myorganization" {}

output "myorganization" {
  value = data.orcasecurity_organization.myorganization.name
}
