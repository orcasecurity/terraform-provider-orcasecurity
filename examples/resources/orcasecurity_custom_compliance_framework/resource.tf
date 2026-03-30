resource "orcasecurity_custom_compliance_framework" "example" {
  name        = "My Custom Framework - Igor tests"
  description = "Custom compliance framework managed by Terraform"

  sections = [
    {
      name = "Access Control"
      tests = [
        {
          rule_id              = "r8ae477067a"
          rule_id_in_framework = "1.1"
        },
        {
          rule_id              = "r8ae477067a"
          rule_id_in_framework = "1.2"
        }
      ]
    }
  ]
}
