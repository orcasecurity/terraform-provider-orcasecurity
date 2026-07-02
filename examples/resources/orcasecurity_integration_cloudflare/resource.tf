# Cloudflare integration
resource "orcasecurity_integration_cloudflare" "example" {
  template_name = "test_cloudflare"
  api_token     = var.cloudflare_api_token
  is_enabled    = true
  is_default    = false
}
