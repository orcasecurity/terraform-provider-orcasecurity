resource "orcasecurity_admission_controller_policy" "baseline" {
  name               = "Baseline policy"
  description        = "Baseline admission controller policy"
  is_active          = true
  enforcement_action = "monitor"
  controls           = [orcasecurity_admission_controller_control.allowed_repos.id]
}
