resource "orcasecurity_shift_left_bitbucket_account" "example" {
  installation_id   = "11111111-1111-1111-1111-111111111111"
  account_id        = "acme-workspace"
  installation_mode = "SCAN_ALL_INCLUDE_FUTURE"
  default_policies  = true

  configuration_settings = {
    pr_summary_comment        = "ONLY_ON_FAILED_ISSUES"
    comments_on_pull_requests = "ONLY_ON_FAILED_ISSUES"
    skip_check_runs           = "ALWAYS"
    config_file_support       = "ENABLED"
  }
}

resource "orcasecurity_shift_left_bitbucket_account" "project_bound" {
  installation_id   = "11111111-1111-1111-1111-111111111111"
  account_id        = "other-workspace"
  installation_mode = "SCAN_ALL_INCLUDE_FUTURE"
  project_id        = "44444444-4444-4444-4444-444444444444"
}
