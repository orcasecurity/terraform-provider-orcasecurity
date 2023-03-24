terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

provider "orcasecurity" {}

data "orcasecurity_rbac_group" "example" {}

resource "orcasecurity_rbac_group" "admin_group" {
  name        = "Admin"
  description = "This group is for admins"
  sso_group   = true
}


output "orca_rbac_groups" {
  value = data.orcasecurity_rbac_group.example
}
