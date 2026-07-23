data "orcasecurity_shift_left_azure_devops_accounts" "all" {}

# Configure PR settings on every current Azure DevOps integrated account.
resource "orcasecurity_shift_left_azure_devops_account" "all" {
  for_each        = { for a in data.orcasecurity_shift_left_azure_devops_accounts.all.accounts : a.id => a }
  installation_id = each.value.installation_id
  account_id      = each.value.account_id

  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}
