// Group with no members: users is optional; use []
resource "orcasecurity_group" "tf-group-empty-members" {
  name        = "Orca Terraform Group (no members)"
  sso_group   = false
  description = "Example when users is omitted or empty"
  users       = []
}
