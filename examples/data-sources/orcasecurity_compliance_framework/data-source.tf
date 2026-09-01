data "orcasecurity_compliance_framework" "gcp_cis" {
  id = "gcp_cis_3.0.0"
}

output "controls" {
  value = data.orcasecurity_compliance_framework.gcp_cis.sections
}
