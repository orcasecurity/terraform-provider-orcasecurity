data "orcasecurity_rbac_roles" "all" {}

# Resolve a role id by exact name (single match required)
locals {
  viewer_role_id = one([
    for r in data.orcasecurity_rbac_roles.all.roles : r.id
    if r.name == "Viewer"
  ])
}

output "role_count" {
  value = length(data.orcasecurity_rbac_roles.all.roles)
}

output "viewer_role_id" {
  value = local.viewer_role_id
}

output "first_five_roles" {
  value = [for i, r in data.orcasecurity_rbac_roles.all.roles : r if i < 5]
}
