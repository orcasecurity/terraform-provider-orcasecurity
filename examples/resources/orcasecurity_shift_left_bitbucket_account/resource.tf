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
