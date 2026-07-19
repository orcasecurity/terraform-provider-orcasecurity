# Bulk, deterministic ordering of several automations as one unit.
#
# The automations below are ordered by a single resource instead of setting the
# `priority` attribute on each one. Terraform assigns priority 1 to the first
# ID, 2 to the second, and so on, in one sequential pass — so a full reorder
# converges in a single apply, which per-automation `priority` cannot guarantee.
#
# Do NOT also set `priority` on these automations: the two mechanisms write the
# same global ordering and will fight.

resource "orcasecurity_automation_v2" "critical" {
  name   = "critical alerts"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({ models = ["Alert"], type = "object_set" })
  }
  alert_dismissal_details = { reason = "critical" }
}

resource "orcasecurity_automation_v2" "high" {
  name   = "high alerts"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({ models = ["Alert"], type = "object_set" })
  }
  alert_dismissal_details = { reason = "high" }
}

resource "orcasecurity_automation_v2" "default" {
  name   = "default alerts"
  status = "enabled"
  filter = {
    sonar_query = jsonencode({ models = ["Alert"], type = "object_set" })
  }
  alert_dismissal_details = { reason = "default" }
}

# The listed automations occupy priorities 1..N in list order. To reorder, just
# rearrange this list and re-apply. Every other automation in the organization
# keeps its relative order below these.
resource "orcasecurity_automation_v2_priority_order" "main" {
  automation_ids = [
    orcasecurity_automation_v2.critical.id, # priority 1
    orcasecurity_automation_v2.high.id,     # priority 2
    orcasecurity_automation_v2.default.id,  # priority 3
  ]
}
