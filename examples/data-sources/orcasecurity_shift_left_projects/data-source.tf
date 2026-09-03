data "orcasecurity_shift_left_projects" "all" {}

# Attach a policy to a subset of projects selected by name.
# To attach to *every* project, do not enumerate IDs from this data source --
# set attach_all_projects = true on the policy instead, so the API resolves the
# set server-side and a project created between plan and apply is not missed.
resource "orcasecurity_shift_left_policy" "malicious_packages_team_a" {
  name                       = "Malicious packages - team A"
  type                       = "malicious_packages"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"

  projects_ids = [
    for p in data.orcasecurity_shift_left_projects.all.projects : p.id
    if startswith(p.name, "team-a/")
  ]
}
