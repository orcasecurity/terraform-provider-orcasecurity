resource "orcasecurity_shift_left_bitbucket_account" "example" {
  installation_id   = "11111111-1111-1111-1111-111111111111"
  account_id        = "22222222-2222-2222-2222-222222222222"
  installation_mode = "SCAN_ALL_INCLUDE_FUTURE"
  default_policies  = true

  configuration_settings = {
    pr_summary_comment        = "ONLY_ON_FAILED_ISSUES"
    comments_on_pull_requests = "ONLY_ON_FAILED_ISSUES"
    skip_check_runs           = "ALWAYS"
    config_file_support       = "ENABLED"
  }
}

# Alternatively, bind the account to a scan-all project instead of policies
# project_id is mutually exclusive with
# default_policies and policies_ids.
resource "orcasecurity_shift_left_bitbucket_account" "project_bound" {
  installation_id   = "11111111-1111-1111-1111-111111111111"
  account_id        = "33333333-3333-3333-3333-333333333333"
  installation_mode = "SCAN_ALL_INCLUDE_FUTURE"
  project_id        = "44444444-4444-4444-4444-444444444444"
}
