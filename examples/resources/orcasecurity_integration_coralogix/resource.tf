# Coralogix integration
resource "orcasecurity_integration_coralogix" "example" {
  template_name = "coralogix_name"
  webhook_url   = "https://coralogix.us"
  api_key       = var.coralogix_api_key
  is_enabled    = true
  is_default    = false

  business_units = [
    "5308642b-f207-4610-a13b-f39c4db4a7a3",
    "d7a3b159-3063-433b-b954-0586ff3e8438",
  ]

  custom_headers = {
    key_header_1 = [{ custom = "value_header_1" }]
  }
}
