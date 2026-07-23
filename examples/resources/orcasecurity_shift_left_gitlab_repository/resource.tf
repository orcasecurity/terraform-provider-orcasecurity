resource "orcasecurity_shift_left_gitlab_repository" "example" {
  installation_id   = "11111111-2222-3333-4444-555555555555"
  gitlab_group_id   = 87654321
  gitlab_project_id = 123456789
  name              = "acme-group/service-api"
  url               = "https://gitlab.com/acme-group/service-api"

  disable_scan_pull_requests = false
  pr_summary_comment         = "ALWAYS"
}
