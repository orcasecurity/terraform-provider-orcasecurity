data "orcasecurity_shift_left_gitlab_repositories" "all" {}

output "gitlab_repository_names" {
  value = [for r in data.orcasecurity_shift_left_gitlab_repositories.all.repositories : r.name]
}
