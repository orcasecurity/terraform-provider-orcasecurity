resource "orcasecurity_shift_left_policy" "iac_baseline" {
  type                       = "iac"
  name                       = "IaC baseline"
  description                = "Managed by Terraform"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"

  iac {
    controls {
      id       = "orca.best_practices.aws_s3_bucket_public_read_prohibited"
      priority = "HIGH"
      disabled = false
    }
  }
}
