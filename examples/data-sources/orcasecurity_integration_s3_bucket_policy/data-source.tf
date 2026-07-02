# Render the bucket policy Orca needs before creating an S3 bucket integration.
data "orcasecurity_integration_s3_bucket_policy" "orca" {
  arn_or_url = "arn:aws:s3:::my-bucket"
  folder     = "folder_value"
}

output "policy_instructions" {
  value = data.orcasecurity_integration_s3_bucket_policy.orca.bucket_policy_instructions
}

output "policy_json" {
  value = data.orcasecurity_integration_s3_bucket_policy.orca.bucket_policy_json
}
