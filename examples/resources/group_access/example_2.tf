// Role id from GET /api/rbac/role (data.orcasecurity_rbac_roles), not orcasecurity_custom_role.
data "orcasecurity_rbac_roles" "all" {}

locals {
  group_access_role_id = one([
    for r in data.orcasecurity_rbac_roles.all.roles : r.id
    if r.name == "Viewer" # exact name from API
  ])
}

resource "orcasecurity_group_access" "builtin_role_on_bu" {
  group_id           = orcasecurity_group.rbac_api_example.id
  role_id            = local.group_access_role_id
  all_cloud_accounts = false
  user_filters       = [orcasecurity_business_unit.rbac_api_example.id]
}

resource "orcasecurity_business_unit" "rbac_api_example" {
  name = "tf-example-bu-for-group-access-api-role"
  filter_data = {
    cloud_providers = ["aws"]
  }
}

resource "orcasecurity_group" "rbac_api_example" {
  name        = "tf-example-group-for-api-role"
  sso_group   = false
  description = "Group for group_access with role from /api/rbac/role"
  users       = [] # set user UUIDs if your org requires at least one member
}
