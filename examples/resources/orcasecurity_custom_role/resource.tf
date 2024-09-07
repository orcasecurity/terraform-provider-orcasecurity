//Custom role
resource "orcasecurity_custom_role" "tf-custom-role-1" {
  name = "custom_role_1"
  permission_groups = [
    "assets.asset.read",
    "auth.tokens.write"
  ]

  description = "simple role with 2 permissions"

}