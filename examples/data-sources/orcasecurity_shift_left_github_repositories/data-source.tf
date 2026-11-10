data "orcasecurity_shift_left_github_repositories" "all" {}

output "github_repository_names" {
  value = [for r in data.orcasecurity_shift_left_github_repositories.all.repositories : r.name]
}
