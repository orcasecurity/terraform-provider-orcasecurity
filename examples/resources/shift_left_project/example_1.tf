// Shift Left project
resource "orcasecurity_shift_left_project" "shift_left_project_1" {
  name             = "Project 1"
  description      = "Project for all repos"
  key              = "project-1"
  default_policies = true
}
