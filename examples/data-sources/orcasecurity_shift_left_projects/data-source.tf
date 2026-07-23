data "orcasecurity_shift_left_projects" "all" {}

# Attach one policy to every current shift-left project in a single apply,
# e.g. a built-in policy previously imported into Terraform state.
resource "orcasecurity_shift_left_policy" "malicious_packages" {
  # ... (imported built-in policy; type/name/etc. come from state) ...

  projects_ids = [for p in data.orcasecurity_shift_left_projects.all.projects : p.id]
}
