// POST /api/rbac/access/user — grant an RBAC role to an existing user, scoped to a business unit.
resource "orcasecurity_user_access" "role_on_bu" {
  user_id            = "{place-user-id-here}" // from the /api/users endpoint
  role_id            = orcasecurity_custom_role.example.id
  all_cloud_accounts = false
  user_filters       = [orcasecurity_business_unit.example.id]
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
