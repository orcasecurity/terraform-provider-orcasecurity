resource "orcasecurity_custom_widget" "my_widget" {
  name               = "My Custom Widget"
  organization_level = true
  extra_params = {
    type                = "donut"
    empty_state_message = "No data found"
    default_size        = "sm"
    is_new              = true
    subtitle            = ""
    description         = ""
    settings = {
      columns = ["alert", "asset"]
      request_params = {
        query = jsonencode({
          keys   = ["Alert"]
          type   = "object_set"
          models = ["Alert"]
        })
        group_by      = ["RiskLevel"]
        group_by_list = ["RiskLevel"]
      }
      field = { name = "RiskLevel", type = "str" }
    }
  }
}
