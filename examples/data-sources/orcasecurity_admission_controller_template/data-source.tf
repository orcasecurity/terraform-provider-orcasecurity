# Look up a template by its internal name (unique).
data "orcasecurity_admission_controller_template" "allowed_repos" {
  name = "k8sallowedrepos"
}

# Or by its display name as shown in the Orca UI.
data "orcasecurity_admission_controller_template" "by_display_name" {
  display_name = "Allowed Container Registries"
}
