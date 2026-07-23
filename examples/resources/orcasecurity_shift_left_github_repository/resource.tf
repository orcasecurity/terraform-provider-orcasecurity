resource "orcasecurity_shift_left_github_repository" "example" {
  installation_id      = "11111111-2222-3333-4444-555555555555"
  github_repository_id = 123456789
  name                 = "acme/service-api"
  url                  = "https://github.com/acme/service-api"
  branch               = "main"

  project_id                = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
  comments_on_pull_requests = "ONLY_ON_FAILED_ISSUES"
  config_file_support       = "ENABLED"
}
