# Manages the org-wide built-in SCM posture policy (always exists; adopt-style).
resource "orcasecurity_shift_left_scm_posture_default_policy" "default" {
  disabled = false

  controls = [
    {
      id       = "branch_protection_enforce_admins"
      priority = "HIGH"
    },
    {
      id       = "repo_forking_enabled"
      disabled = true
    },
  ]
}
