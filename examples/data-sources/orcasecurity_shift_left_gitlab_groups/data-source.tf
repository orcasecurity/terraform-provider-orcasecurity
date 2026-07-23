data "orcasecurity_shift_left_gitlab_groups" "all" {}

# Configure PR/MR settings on every current GitLab integrated group.
resource "orcasecurity_shift_left_gitlab_group" "all" {
  for_each        = { for g in data.orcasecurity_shift_left_gitlab_groups.all.groups : g.id => g }
  installation_id = each.value.installation_id
  group_id        = each.value.group_id

  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}
