terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

provider "orcasecurity" {}

data "orcasecurity_rbac_group" "example" {}

output "orca_rbac_groups" {
  value = data.orcasecurity_rbac_group.example
}
