data "orcasecurity_shift_left_bitbucket_repositories" "all" {}

output "bitbucket_repository_names" {
  value = [for r in data.orcasecurity_shift_left_bitbucket_repositories.all.repositories : r.name]
}
