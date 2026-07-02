# Zscaler ZPA integration
resource "orcasecurity_integration_zscaler_zpa" "example" {
  template_name = "template_name"
  vanity_domain = "vanity_domain"
  client_id     = var.zscaler_client_id
  client_secret = var.zscaler_client_secret
  is_enabled    = true
  is_default    = false
}
