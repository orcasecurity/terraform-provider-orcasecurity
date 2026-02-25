# Donut-type widget: cloud KMS keys grouped by type.
# Uses group_by so the API receives group_by[] for correct aggregation.
resource "orcasecurity_custom_widget" "kms_by_type" {
  name               = "KMS Keys by Type"
  organization_level = true
  extra_params = {
    type                = "donut"
    empty_state_message = "Widget query returned no data"
    default_size        = "sm"
    is_new              = true
    subtitle            = "Cloud KMS keys by type"
    description         = "Distribution of KMS keys across cloud providers"
    settings = {
      request_params = {
        query = jsonencode({
          type = "object_set"
          models = [
            "AliCloudKmsKey",
            "AwsCloudHsmV2Cluster",
            "AwsCloudHsmV2Hsm",
            "AwsKmsKey",
            "AzureKeyVault",
            "AzureKeyVaultKey",
            "GcpKmsKey",
            "OciKmsKey",
            "OciVault",
            "TencentCloudKmsKey"
          ]
        })
        group_by          = ["Type"]
        group_by_list     = ["CloudAccount.Name"]
        start_at_index    = 0
        order_by          = ["-COUNT"]
        limit             = 1000
        enable_pagination = false
      }
      field = {
        name = "Type"
        type = "str"
      }
    }
  }
}
