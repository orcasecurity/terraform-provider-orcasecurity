# custom sensitive data identifier matching 9-digit account numbers
resource "orcasecurity_sensitive_data_identifier" "account_number" {
  title        = "Internal Account Number"
  details      = "Detects internal 9-digit account numbers."
  category     = "PII"
  sub_category = "Financial"
  properties = {
    conditions = [
      { value = "ACC-[0-9]{9}" }
    ]
    detection_types = ["text", "db"]
    sensitivity     = "high"
    significance    = "major"
    keywords        = ["account", "acct"]
    text_threshold  = 2
  }
}

# disabled identifier with default detection types
resource "orcasecurity_sensitive_data_identifier" "legacy_token" {
  title        = "Legacy API Token"
  details      = "Detects legacy API token format."
  category     = "SECRET"
  sub_category = "Credentials"
  enabled      = false
  properties = {
    conditions = [
      { value = "legacy_[a-f0-9]{32}" }
    ]
  }
}
