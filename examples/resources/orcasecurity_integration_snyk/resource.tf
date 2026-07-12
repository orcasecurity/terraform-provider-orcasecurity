# Snyk integration
resource "orcasecurity_integration_snyk" "example" {
  template_name = "display_name"
  api_token     = var.snyk_api_token
  # Match what the Orca UI shows in the region dropdown:
  #   "United States"    -> US
  #   "United States 2"  -> US2
  #   "European Union"   -> EU
  #   "Australia"        -> AU
  region     = "US"
  is_enabled = true
  is_default = false
}
