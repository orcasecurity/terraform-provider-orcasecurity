resource "orcasecurity_custom_alert" "myalert" {
  name            = "Alert name"
  description     = "Alert description"
  rule            = "ActivityLogDetection"
  score           = 5.5
  category        = "Best practices"
  allow_adjusting = false
}

// with remediation text
resource "orcasecurity_custom_alert" "myalert" {
  name            = "Alert name"
  description     = "Alert description"
  rule            = "ActivityLogDetection"
  score           = 5.5
  category        = "Best practices"
  allow_adjusting = false
  remediation_text = {
    enable = true
    text   = "This is a remediation text."
  }
}

// with custom compliance frameworks
resource "orcasecurity_custom_alert" "myalert" {
  name            = "Alert name"
  description     = "Alert description"
  rule            = "ActivityLogDetection"
  score           = 5.5
  category        = "Best practices"
  allow_adjusting = false
  compliance_frameworks = [
    { name = "framework name", section = "framework section", priority = "low" }
  ]
}
