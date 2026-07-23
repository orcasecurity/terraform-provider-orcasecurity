data "orcasecurity_shift_left_github_installations" "all" {}

# Configure PR settings on every current GitHub installation.
resource "orcasecurity_shift_left_github_installation" "all" {
  for_each        = { for i in data.orcasecurity_shift_left_github_installations.all.installations : i.id => i }
  installation_id = each.value.id

  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}
