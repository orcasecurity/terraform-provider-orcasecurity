// POST /api/rbac/access/user — grant an RBAC role to an existing user, scoped to a business unit.
data "orcasecurity_users" "all" {}

resource "orcasecurity_user_access" "role_on_bu" {
  user_id            = one([for u in data.orcasecurity_users.all.users : u.user_id if u.email == "jane@example.com"])
  role_id            = orcasecurity_custom_role.example.id
  all_cloud_accounts = false
  user_filters       = [orcasecurity_business_unit.example.id]
}

// Built-in Shift Left roles belong in the Shift Left global scope, not in all_cloud_accounts.
data "orcasecurity_rbac_roles" "all" {}

resource "orcasecurity_user_access" "shiftleft_user_on_all_projects" {
  user_id            = one([for u in data.orcasecurity_users.all.users : u.user_id if u.email == "dev@example.com"])
  role_id            = one([for r in data.orcasecurity_rbac_roles.all.roles : r.id if r.name == "Shiftleft User"])
  all_cloud_accounts = false
  shiftleft_global   = true
}

resource "orcasecurity_custom_role" "example" {
  name              = "tf_example_read_only"
  description       = "Example role for user_access demo"
  permission_groups = ["assets.asset.read"]
}

resource "orcasecurity_business_unit" "example" {
  name = "tf-example-bu-for-user-access"
  filter_data = {
    cloud_providers = ["aws"]
  }
}
