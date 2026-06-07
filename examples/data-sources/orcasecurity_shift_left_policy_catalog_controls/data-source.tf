data "orcasecurity_shift_left_policy_catalog_controls" "iac" {
  type = "iac"
}

output "iac_control_ids" {
  value = [for c in data.orcasecurity_shift_left_policy_catalog_controls.iac.controls : c.id]
}
