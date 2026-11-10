data "orcasecurity_shift_left_gitlab_groups" "all" {}

# Configure PR/MR settings on every current GitLab integrated group.
#
# adopt_existing is required for a fleet-wide for_each: every group returned by the data source
# already exists in Orca, so `apply` takes over each live integration and a later `destroy` would
# DE-INTEGRATE it, removing repositories and settings that may have been configured outside
# Terraform. Without it, `apply` refuses any group that already has integrated repositories.
# To manage these groups without a takeover write, `terraform import` them individually instead.
resource "orcasecurity_shift_left_gitlab_group" "all" {
  for_each        = { for g in data.orcasecurity_shift_left_gitlab_groups.all.groups : g.id => g }
  installation_id = each.value.installation_id
  gitlab_group_id = each.value.gitlab_group_id
  adopt_existing  = true

  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}
