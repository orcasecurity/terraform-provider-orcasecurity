resource "orcasecurity_automation_v2_priority_order" "main" {
  automation_ids = [
    orcasecurity_automation_v2.critical.id, # priority 1
    orcasecurity_automation_v2.high.id,     # priority 2
    orcasecurity_automation_v2.default.id,  # priority 3
  ]
}
