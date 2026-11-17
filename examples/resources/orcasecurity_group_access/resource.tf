// POST /api/rbac/access/group — role on a group with BU scope via user_filters.
resource "orcasecurity_group_access" "role_on_bu" {
  group_id           = orcasecurity_group.example.id
  role_id            = orcasecurity_custom_role.example.id
  all_cloud_accounts = false
  user_filters       = [orcasecurity_business_unit.example.id]
}

// With no scope set, a built-in Shift Left role defaults to the Shift Left global scope
// (shiftleft_global = true); any other role defaults to all_cloud_accounts = true.
// An explicit all_cloud_accounts = true grants a built-in Shift Left role org-wide. Older configs
// set it because the attribute used to be required: replace it with shiftleft_global = true,
// or remove the scope, to keep these roles in the Shift Left global scope.
data "orcasecurity_rbac_roles" "all" {}

resource "orcasecurity_group_access" "shiftleft_user_on_all_projects" {
  group_id = orcasecurity_group.example.id
  role_id  = one([for r in data.orcasecurity_rbac_roles.all.roles : r.id if r.name == "Shiftleft User"])
}

resource "orcasecurity_custom_role" "example" {
  name              = "tf_example_read_only"
  description       = "Example role for group_access demo"
  permission_groups = ["assets.asset.read"]
}

resource "orcasecurity_business_unit" "example" {
  name = "tf-example-bu-for-group-access"
  filter_data = {
    cloud_providers = ["aws"]
  }
}

resource "orcasecurity_group" "example" {
  name        = "tf-example-group-for-access"
  sso_group   = false
  description = "Group used to demo orcasecurity_group_access"
  users       = ["{place-user-id-here}"]
}
