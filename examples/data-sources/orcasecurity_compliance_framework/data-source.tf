data "orcasecurity_compliance_framework" "cis_aws" {
  id = "cis_aws_foundations_1_4_0"
}

output "controls" {
  value = data.orcasecurity_compliance_framework.cis_aws.sections
}
