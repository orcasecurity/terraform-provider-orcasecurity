# Akamai integration
resource "orcasecurity_integration_akamai" "example" {
  template_name = "cred_name"
  host          = "akab-xxxxxxxx.luna.akamaiapis.net"
  access_token  = var.akamai_access_token
  client_token  = var.akamai_client_token
  client_secret = var.akamai_client_secret
  is_enabled    = true
  is_default    = false
}
