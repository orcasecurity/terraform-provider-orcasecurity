# Terraform Cloud integration
resource "orcasecurity_integration_terraform_cloud" "example" {
  template_name = "display_name"
  api_url       = "https://app.terraform.io"
  api_token     = var.terraform_cloud_api_token
  is_enabled    = true
  is_default    = false
}
