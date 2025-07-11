# Example 1: Fetch cloud account by name (base name without ID suffix)
data "orcasecurity_cloud_account" "test_by_name" {
  name = "test-account-name"
}

# Example 2: Fetch cloud account by full name (including ID suffix)
data "orcasecurity_cloud_account" "example_by_full_name" {
  name = "test-account-name (BB5EDED2-8EA3-4608-8BE3-FD0EE8B3F644)"
}

# Example 3: Fetch cloud account by cloud vendor ID
data "orcasecurity_cloud_account" "test_by_vendor_id" {
  cloud_vendor_id = "BB5EDED2-8EA3-4608-8BE3-FD0EE8B3F644"
}

# Output examples showing how to access the data
output "cloud_account_info" {
  value = {
    # Basic information
    account_id  = data.orcasecurity_cloud_account.example_by_name.cloud_account_id
    name        = data.orcasecurity_cloud_account.example_by_name.name
    description = data.orcasecurity_cloud_account.example_by_name.description
    status      = data.orcasecurity_cloud_account.example_by_name.cloud_account_status

    # Provider information
    cloud_provider     = data.orcasecurity_cloud_account.example_by_name.cloud_provider
    provider_id        = data.orcasecurity_cloud_account.example_by_name.cloud_provider_id
    vendor_id          = data.orcasecurity_cloud_account.example_by_name.cloud_vendor_id
    provider_partition = data.orcasecurity_cloud_account.example_by_name.cloud_provider_partition

    # Azure-specific fields (when applicable)
    azure_tenant_id       = data.orcasecurity_cloud_account.example_by_name.azure_tenant_id
    azure_subscription_id = data.orcasecurity_cloud_account.example_by_name.azure_subscription_id

    # AWS-specific fields (when applicable)
    aws_role_arn     = data.orcasecurity_cloud_account.example_by_name.aws_role_arn
    role_external_id = data.orcasecurity_cloud_account.example_by_name.role_external_id

    # GCP-specific fields (when applicable)
    gcp_organization_id = data.orcasecurity_cloud_account.example_by_name.gcp_organization_id

    # Account configuration
    account_type    = data.orcasecurity_cloud_account.example_by_name.type
    is_management   = data.orcasecurity_cloud_account.example_by_name.is_mgmt
    scan_mode       = data.orcasecurity_cloud_account.example_by_name.scan_mode
    scan_in_account = data.orcasecurity_cloud_account.example_by_name.scan_inaccount

    # Timestamps
    created_time = data.orcasecurity_cloud_account.example_by_name.created_time
  }
}

# Example of using cloud account data in other resources
output "azure_tenant_for_other_resources" {
  description = "Azure tenant ID that can be used in other Azure resources"
  value       = data.orcasecurity_cloud_account.example_by_name.azure_tenant_id
}

output "cloud_provider_type" {
  description = "Cloud provider type for conditional logic"
  value       = data.orcasecurity_cloud_account.example_by_name.cloud_provider
}