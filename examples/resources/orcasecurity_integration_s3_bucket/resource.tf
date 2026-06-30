# --------------------------------------------------------------------------------------------
# Render the bucket policy. This call hits Orca's `/api/settings` endpoint only — no
# integration is created here, no connectivity check is run. Safe to apply on its own.
# --------------------------------------------------------------------------------------------
data "orcasecurity_integration_s3_bucket_policy" "orca" {
  arn_or_url = "arn:aws:s3:::my-bucket"
  folder     = "folder_value"
}

# --------------------------------------------------------------------------------------------
# Option A — bucket policy managed outside Terraform.
# Output the JSON, hand it off to whoever owns the bucket. Create the integration in a
# follow-up apply once they confirm the policy is attached.
# --------------------------------------------------------------------------------------------
output "orca_bucket_policy_instructions" {
  value = data.orcasecurity_integration_s3_bucket_policy.orca.bucket_policy_instructions
}

output "orca_bucket_policy_json" {
  value = data.orcasecurity_integration_s3_bucket_policy.orca.bucket_policy_json
}

# --------------------------------------------------------------------------------------------
# Option B — bucket policy managed in this workspace. Feed the policy straight into
# aws_s3_bucket_policy and force the integration to wait for it via depends_on.
# --------------------------------------------------------------------------------------------
# resource "aws_s3_bucket_policy" "orca" {
#   bucket = aws_s3_bucket.target.id
#   policy = data.orcasecurity_integration_s3_bucket_policy.orca.bucket_policy_json
# }

resource "orcasecurity_integration_s3_bucket" "example" {
  template_name = "s3_bucket_name"
  arn_or_url    = "arn:aws:s3:::my-bucket"
  folder        = "folder_value"
  is_enabled    = true
  is_default    = false

  # Uncomment with Option B so Orca's connectivity check only runs after the policy is live.
  # depends_on = [aws_s3_bucket_policy.orca]
}
