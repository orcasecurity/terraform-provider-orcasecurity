data "orcasecurity_shift_left_azure_devops_repositories" "all" {}

output "azure_devops_repository_names" {
  value = [for r in data.orcasecurity_shift_left_azure_devops_repositories.all.repositories : r.name]
}
