// POST /api/user_invites — invite a user and grant an RBAC role scoped to a business unit.
resource "orcasecurity_add_users" "invite_with_role" {
  email              = "new.user@example.com"
  role_id            = orcasecurity_custom_role.example.id
  all_cloud_accounts = false
  user_filters       = [orcasecurity_business_unit.example.id]
}

// Invite a user and add them to one or more groups instead of a role.
resource "orcasecurity_add_users" "invite_with_groups" {
  email  = "teammate@example.com"
  groups = [orcasecurity_group.example.id]
}

resource "orcasecurity_custom_role" "example" {
  name              = "tf_example_read_only"
  description       = "Example role for add_users demo"
  permission_groups = ["assets.asset.read"]
}

resource "orcasecurity_business_unit" "example" {
  name = "tf-example-bu-for-add-users"
  filter_data = {
    cloud_providers = ["aws"]
  }
}

resource "orcasecurity_group" "example" {
  name        = "tf-example-group-for-add-users"
  sso_group   = false
  description = "Group used to demo orcasecurity_add_users"
}
