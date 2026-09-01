data "orcasecurity_compliance_frameworks" "active_custom" {
  custom = true
  active = true
}

data "orcasecurity_compliance_frameworks" "orca" {
  type   = "Orca Frameworks"
  search = "best practices"
}

data "orcasecurity_compliance_frameworks" "pci" {
  version_agnostic_display_name = "PCI DSS"
}

output "active_custom_ids" {
  value = [for f in data.orcasecurity_compliance_frameworks.active_custom.frameworks : f.id]
}
