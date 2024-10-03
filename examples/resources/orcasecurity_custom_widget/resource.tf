//custom widget resource
//custom widget resource
resource "orcasecurity_custom_widget" "tf-custom-widget-1" {
  name               = "Custom Widget 45"
  organization_level = true
  view_type          = "customs_widgets"
  extra_params = {
    type                = "PIE_CHART_SINGLE",
    category            = "Custom",
    empty_state_message = "No data found",
    default_size        = "sm",
    is_new              = true,
    title               = "Custom Widget 45",
    subtitle            = "Sample subtitle",
    description         = "Sample description",
    settings = {
      request_params = {
        query = jsonencode({
          "models" : [
            "GcpApiKey"
          ],
          "type" : "object_set"
        })
        group_by : [
          "Type"
        ],
        group_by_list = [
          "CloudAccount.Name"
        ]
      }
      field = {
        name = "CloudAccount.Name",
        type = "str"
      }
    }
  }
}