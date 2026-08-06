data "orcasecurity_shift_left_gitlab_installations" "all" {}

output "gitlab_installation_ids" {
  value = [for i in data.orcasecurity_shift_left_gitlab_installations.all.installations : i.id]
}
