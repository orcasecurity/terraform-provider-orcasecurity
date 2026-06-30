# Splunk HEC integration
resource "orcasecurity_integration_splunk" "example" {
  template_name          = "splunk_test"
  url                    = "https://prd-p-splunk.splunkcloud.com:8088/services/collector/event"
  token                  = var.splunk_hec_token
  allow_self_signed_cert = true
  is_enabled             = true
  is_default             = false
}
