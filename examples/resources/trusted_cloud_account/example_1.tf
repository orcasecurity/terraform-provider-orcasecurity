# trusted cloud account
resource "orcasecurity_trusted_cloud_account" "example" {
  account_name      = "Vendor account"
  description       = "This is the AWS account for a security vendor we use. This account allows them to read risks in our cloud environment."
  cloud_provider    = "aws"
  cloud_provider_id = "123412341234"
}
