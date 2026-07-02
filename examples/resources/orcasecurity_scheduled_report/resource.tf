# weekly alerts report based on a discovery (sonar) query, delivered by email
resource "orcasecurity_scheduled_report" "weekly_alerts" {
  name              = "Weekly open alerts"
  type              = "alerts_svl"
  format            = "csv"
  recurrence        = "weekly"
  first_report_date = "2026-07-01T13:00:00Z"
  export_time       = "13:00:00"

  recipients_emails    = ["security-team@example.com"]
  custom_email_subject = "Weekly Orca alerts report"
  custom_email_content = "Attached is the weekly alerts report."

  sonar_query = jsonencode({
    models = ["Alert"]
    type   = "object_set"
    with = {
      operator = "and"
      type     = "operation"
      values = [
        {
          key      = "RiskLevel"
          values   = ["critical", "high", "medium"]
          type     = "str"
          operator = "in"
        },
        {
          key      = "Status"
          values   = ["open", "in_progress"]
          type     = "str"
          operator = "in"
        }
      ]
    }
  })

  sonar_query_params = jsonencode({
    "additionalModels[]" = ["CloudAccount", "BusinessUnits.Name", "Inventories", "Inventory"]
    "order_by[]"         = ["-OrcaScore"]
    "group_by[]"         = ["AlertType"]
    max_tier             = 5
    full_graph_fetch = {
      enabled        = true
      limit_children = 20
    }
    sonar_filter = {}
  })

  columns = [
    "OrcaScore", "Title", "Category", "Inventory.Name",
    "CloudAccount.Name", "CreatedAt", "LastSeen", "AlertId",
  ]

  # Compression is set via the typed attribute; other extras go in `config`.
  compression = ".zip"

  config = jsonencode({
    buIds = []
  })
}

# daily discovery report uploaded to a connected S3 bucket
resource "orcasecurity_scheduled_report" "daily_assets" {
  name              = "Daily internet-facing assets export"
  type              = "discovery_svl"
  format            = "csv"
  recurrence        = "daily"
  first_report_date = "2026-07-01T00:00:00Z"
  export_time       = "00:00:00"

  sonar_query = jsonencode({
    models = ["Inventory"]
    type   = "object_set"
    with = {
      operator = "and"
      type     = "operation"
      values = [
        {
          key      = "IsInternetFacing"
          values   = [true]
          type     = "bool"
          operator = "eq"
        }
      ]
    }
  })

  share_to_bucket = true
  bucket          = "my-connected-bucket-template"
}
