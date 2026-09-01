data "orcasecurity_compliance_frameworks" "active_custom" {
  custom = true
  active = true
}

data "orcasecurity_compliance_frameworks" "orca" {
  type   = "Orca Frameworks"
  search = "best practices"
}

output "active_custom_ids" {
  value = [for f in data.orcasecurity_compliance_frameworks.active_custom.frameworks : f.id]
}
