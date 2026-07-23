resource "orcasecurity_shift_left_github_installation" "example" {
  installation_id   = "11111111-1111-1111-1111-111111111111"
  installation_mode = "SCAN_ALL_INCLUDE_FUTURE"
  default_policies  = true

  configuration_settings = {
    pr_summary_comment        = "ONLY_ON_FAILED_ISSUES"
    comments_on_pull_requests = "ONLY_ON_FAILED_ISSUES"
    skip_check_runs           = "ALWAYS"
    config_file_support       = "ENABLED"
  }
}

# Alternatively, bind the installation to a scan-all project instead of policies
# project_id is mutually exclusive with
# default_policies and policies_ids.
resource "orcasecurity_shift_left_github_installation" "project_bound" {
  installation_id   = "55555555-5555-5555-5555-555555555555"
  installation_mode = "SCAN_ALL_INCLUDE_FUTURE"
  project_id        = "44444444-4444-4444-4444-444444444444"
}
