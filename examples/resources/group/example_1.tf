data "orcasecurity_users" "all" {}

//Group
resource "orcasecurity_group" "tf-group-1" {
  name = "Orca Terraform Group 1"

  sso_group   = true
  description = "string"
  users = [
    one([for u in data.orcasecurity_users.all.users : u.user_id if u.email == "jane@example.com"])
  ]
}
