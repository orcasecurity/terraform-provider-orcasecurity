resource "orcasecurity_rbac_group" "admin_group" {
  name        = "Admin"
  description = "This group is for admins."
  sso_group   = true
}
