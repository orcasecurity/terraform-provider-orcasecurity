data "orcasecurity_shift_left_projects" "all" {}

# Attach one policy to every current shift-left project in a single apply.
# projects_ids is recomputed on each plan, so a project added in Orca is picked up
# by the next apply.
resource "orcasecurity_shift_left_policy" "malicious_packages" {
  name                       = "Malicious packages - all projects"
  type                       = "malicious_packages"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"

  projects_ids = [for p in data.orcasecurity_shift_left_projects.all.projects : p.id]
}
