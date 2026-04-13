resource "orcasecurity_custom_compliance_framework" "example" {
  name        = "My Custom Framework"
  description = "Custom compliance framework managed by Terraform"

  sections = [
    {
      name = "Access Control"
      tests = [
        {
          rule_id              = "your_rule_id"
          rule_id_in_framework = "1.1"
        },
        {
          rule_id              = "your_rule_id"
          rule_id_in_framework = "1.2"
        }
      ]
    },
    {
      name = "Data Protection"
      tests = [
        {
          rule_id              = "your_rule_id"
          rule_id_in_framework = "2.1"
        }
      ]
    }
  ]
}
