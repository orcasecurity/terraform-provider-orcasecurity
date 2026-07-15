# Assign a policy to specific Kubernetes clusters.
resource "orcasecurity_admission_controller_policy_assignment" "production" {
  name        = "Production clusters"
  description = "Baseline policy on production clusters"
  clusters    = ["11111111-2222-3333-4444-555555555555"]
  policy_ids  = [orcasecurity_admission_controller_policy.baseline.id]
}

# Or assign to every cluster in the organization.
resource "orcasecurity_admission_controller_policy_assignment" "org_wide" {
  name              = "Organization wide"
  full_organization = true
  policy_ids        = [orcasecurity_admission_controller_policy.baseline.id]
}
