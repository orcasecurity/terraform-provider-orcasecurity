data "orcasecurity_shift_left_bitbucket_installations" "all" {}

output "bitbucket_installation_ids" {
  value = [for i in data.orcasecurity_shift_left_bitbucket_installations.all.installations : i.id]
}
