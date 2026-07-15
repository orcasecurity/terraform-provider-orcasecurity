# custom sensitive data identifier matching 9-digit account numbers.
# The regex must start and end with a boundary and contain a named
# capturing group called `secret`; sub_category must be a valid Orca
# catalog sub-category.
resource "orcasecurity_sensitive_data_identifier" "account_number" {
  title        = "Internal Account Number"
  details      = "Detects internal 9-digit account numbers."
  category     = "PII"
  sub_category = "Financial Account Numbers"
  properties = {
    conditions = [
      { value = "\\b(?P<secret>ACC-[0-9]{9})\\b" }
    ]
    detection_types = ["text", "db"]
    sensitivity     = "high"
    significance    = "Major"
    keywords        = ["account", "acct"]
    text_threshold  = 2
  }
}

# disabled identifier with default detection types
resource "orcasecurity_sensitive_data_identifier" "legacy_token" {
  title        = "Legacy API Token"
  details      = "Detects legacy API token format."
  category     = "SECRET"
  sub_category = "API Keys and Tokens"
  enabled      = false
  properties = {
    conditions = [
      { value = "\\b(?P<secret>legacy_[a-f0-9]{32})\\b" }
    ]
  }
}
