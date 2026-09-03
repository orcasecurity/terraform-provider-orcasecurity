data "orcasecurity_shift_left_azure_devops_installations" "all" {}

output "azure_devops_installation_ids" {
  value = [for i in data.orcasecurity_shift_left_azure_devops_installations.all.installations : i.id]
}
