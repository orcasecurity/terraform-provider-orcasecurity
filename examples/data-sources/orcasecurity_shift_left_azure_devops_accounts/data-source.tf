data "orcasecurity_shift_left_azure_devops_accounts" "all" {}

# Configure PR settings on every current Azure DevOps integrated account.
#
# adopt_existing is required for a fleet-wide for_each: every account returned by the data source
# already exists in Orca, so `apply` takes over each live integration and a later `destroy` would
# DE-INTEGRATE it, removing repositories and settings that may have been configured outside
# Terraform. Without it, `apply` refuses any account that already has integrated repositories.
# To manage these accounts without a takeover write, `terraform import` them individually instead.
resource "orcasecurity_shift_left_azure_devops_account" "all" {
  for_each        = { for a in data.orcasecurity_shift_left_azure_devops_accounts.all.accounts : a.id => a }
  installation_id = each.value.installation_id
  account_name    = each.value.account_name
  adopt_existing  = true

  configuration_settings = {
    pr_summary_comment = "ONLY_ON_FAILED_ISSUES"
  }
}
