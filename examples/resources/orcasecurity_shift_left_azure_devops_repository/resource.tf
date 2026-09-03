resource "orcasecurity_shift_left_azure_devops_repository" "example" {
  installation_id     = "11111111-2222-3333-4444-555555555555"
  account_name        = "acme-org"
  azure_project_id    = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
  azure_repository_id = "ffffffff-0000-1111-2222-333333333333"
  name                = "acme-org/platform/service-api"
  url                 = "https://dev.azure.com/acme-org/platform/_git/service-api"

  disabled            = false
  config_file_support = "ENABLED"
}
