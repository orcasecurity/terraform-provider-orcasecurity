resource "orcasecurity_custom_compliance_framework" "example" {
  name        = "My Custom Framework"
  description = "Custom compliance framework managed by Terraform"

  sections = [
    {
      name = "Access Control"
      tests = [
        {
          rule_id              = "rc7bcf3b77f"
          rule_id_in_framework = "1"
        },
        {
          rule_id              = "rc7bcf3b77f"
          rule_id_in_framework = "2"
        }
      ]
    },
    {
      name = "Data Protection"
      tests = [
        {
          rule_id              = "rc7bcf3b77f"
          rule_id_in_framework = "3"
        }
      ]
    }
  ]
}
