# Template that defines how Orca alerts map to Monday.com board columns and how
# alert/ticket status changes sync between Orca and Monday.
resource "orcasecurity_integration_monday_template" "demo" {
  template_name = "monday-template-name"
  resource_id   = "49f9b117-e21b-4a70-9763-fc3d44d91e6e"
  workspace_id  = "3709069"
  board_id      = "1827821929"
  group_id      = "topics"

  # List values: a bare string pulls an Orca alert field; an object is a literal
  # (`{ value = ... }`, `{ custom = ... }`, or a person assignment). Non-list values pass
  # through as-is.
  mapping_json = jsonencode({
    long_text_mkn8v2sp = ["alert_id", "asset_name"]
    text               = ["asset_category", "source"]
    numbers_mkn8zhtt   = { custom = "5" }
    person             = [{ value = { id = "66396150", kind = "person" } }]
    status_14          = { value = "0" }
  })

  alert_status_mapping_json  = jsonencode({ snoozed = "1" })
  ticket_status_mapping_json = jsonencode({ "2" = { status = "dismissed" } })

  business_units = [
    "a411f20b-0276-438c-a9d5-938c48a40957",
    "930c1e5e-6b2e-4881-a393-33491e758144",
  ]

  is_enabled = true
  is_default = false
}
