resource "orcasecurity_custom_widget" "test_donut" {
  name               = "test_donut"
  organization_level = true
  extra_params = {
    type                = "donut"
    empty_state_message = "No Data Found"
    default_size        = "sm"
    is_new              = true
    subtitle            = "test_donut"
    description         = "test_donut"
    settings = {
      field = {
        name = "Type"
        type = "str"
      }
      request_params = {
        query = jsonencode({
          models = [
            "AliCloudOssBucket", "AwsS3Bucket", "AwsS3GlacierVault",
            "AzureBlobStorage", "AzureStorageAccount", "GcpStorageBucket",
            "OciStorageBucket", "Storage", "AwsS3AccessPoint",
            "AwsS3MultiRegionAccessPoint", "TencentCloudCosBucket", "LinodeBucket"
          ]
          type = "object_set"
        })
        group_by = ["Type"]
      }
    }
  }
}

resource "orcasecurity_custom_widget" "test_table" {
  name               = "test_table"
  organization_level = true
  extra_params = {
    type                = "table"
    empty_state_message = "No Data Found"
    default_size        = "sm"
    is_new              = true
    subtitle            = "test_table"
    description         = "test_table"
    settings = {
      columns = ["$overview", "CloudAccount", "CloudAccount.AutoRemediationEnabled"]
      request_params = {
        query = jsonencode({
          models = [
            "AliCloudOssBucket", "AwsS3Bucket", "AwsS3GlacierVault",
            "AzureBlobStorage", "AzureStorageAccount", "GcpStorageBucket",
            "OciStorageBucket", "Storage", "AwsS3AccessPoint",
            "AwsS3MultiRegionAccessPoint", "TencentCloudCosBucket", "LinodeBucket"
          ]
          type = "object_set"
        })
        group_by          = ["Type"]
        order_by          = ["-OrcaScore"]
        limit             = 10
        start_at_index    = 0
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "test_metric" {
  name               = "test_metric"
  organization_level = true
  extra_params = {
    type                = "metric" # maps to ICON_GRID
    empty_state_message = "No Data Found"
    default_size        = "sm"
    is_new              = true
    subtitle            = "test_metric"
    description         = "test_metric"
    settings = {
      # no columns, no field
      request_params = {
        query = jsonencode({
          models = [
            "AliCloudOssBucket", "AwsS3Bucket", "AwsS3GlacierVault",
            "AzureBlobStorage", "AzureStorageAccount", "GcpStorageBucket",
            "OciStorageBucket", "Storage", "AwsS3AccessPoint",
            "AwsS3MultiRegionAccessPoint", "TencentCloudCosBucket", "LinodeBucket"
          ]
          type = "object_set"
        })
        group_by = ["Type"]
      }
    }
  }
}

resource "orcasecurity_custom_widget" "test_comparison" {
  name               = "test_comparison"
  organization_level = true
  extra_params = {
    type                = "comparison"
    empty_state_message = "No Data Found"
    default_size        = "sm"
    is_new              = true
    subtitle            = "test_comparison"
    description         = "test_comparison"

    widget_extra_params = {
      field = {
        name = "[Not specified]"
        type = "str"
      }
      values_format = "default"
      default_mapper = jsonencode({
        main       = { color = "var(--theme-color-dataviz-negative)" }
        comparison = { color = "var(--theme-color-dataviz-positive)" }
      })
    }

    settings = {
      request_params_list = [
        {
          id    = "main"
          title = "main"
          query = jsonencode({
            models = ["AliCloudOssBucket", "AwsS3Bucket", "Storage"]
            type   = "object_set"
          })
          group_by = ["Type"]
        },
        {
          id    = "comparison"
          title = "comparison"
          query = jsonencode({
            models = ["AliCloudOssBucket", "AwsS3Bucket", "Storage"]
            type   = "object_set"
          })
          group_by = ["Type"]
        }
      ]
    }
  }
}
