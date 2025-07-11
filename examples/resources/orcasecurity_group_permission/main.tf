resource "orcasecurity_group_permission" "test" {
  group_id           = var.group_id
  role_id            = var.role_id
  all_cloud_accounts = var.all_cloud_accounts
  cloud_accounts     = var.cloud_accounts
  business_units     = var.business_units
  shiftleft_projects = var.shiftleft_projects
}
