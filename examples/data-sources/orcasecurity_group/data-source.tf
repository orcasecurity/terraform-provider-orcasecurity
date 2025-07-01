# Example 1: Fetch group by name
data "orcasecurity_group" "admin_group" {
  name = "Administrators"
}

# Example 2: Fetch a custom group
data "orcasecurity_group" "dev_team" {
  name = "Development Team"
}

# Output examples showing how to access the data
output "group_info" {
  value = {
    # Basic information
    group_id    = data.orcasecurity_group.admin_group.id
    name        = data.orcasecurity_group.admin_group.name
    description = data.orcasecurity_group.admin_group.description

    # Group configuration
    is_sso_group = data.orcasecurity_group.admin_group.sso_group

    # Users in the group
    user_count = length(data.orcasecurity_group.admin_group.users)
    users      = data.orcasecurity_group.admin_group.users
  }
}

# Example of using group data for conditional logic
output "group_type" {
  description = "Determine if this is an SSO group or internal group"
  value       = data.orcasecurity_group.admin_group.sso_group ? "SSO Group" : "Internal Group"
}

# Example of accessing specific users
output "first_user_in_group" {
  description = "First user ID in the group (if any users exist)"
  value       = length(data.orcasecurity_group.admin_group.users) > 0 ? tolist(data.orcasecurity_group.admin_group.users)[0] : "No users"
}