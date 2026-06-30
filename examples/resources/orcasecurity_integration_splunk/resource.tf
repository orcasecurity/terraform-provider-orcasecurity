# Splunk HEC integration
resource "orcasecurity_integration_splunk" "example" {
  template_name          = "Test_Dor"
  url                    = "https://prd-p-o3tuy.splunkcloud.com:8088/services/collector/event"
  token                  = var.splunk_hec_token
  allow_self_signed_cert = true
  is_enabled             = true
  is_default             = false
}
