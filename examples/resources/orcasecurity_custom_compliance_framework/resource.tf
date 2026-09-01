# Build a custom framework from controls on a built-in one. The single-framework
# data source returns sections[].tests[].rule_id; omit rule_id_in_framework and
# the provider derives it as <section-id>.<1-based index>, matching the Orca UI.
data "orcasecurity_compliance_framework" "source" {
  id = "gcp_cis_3.0.0"
}

locals {
  source_tests = flatten([
    for s in data.orcasecurity_compliance_framework.source.sections : concat(
      s.tests != null ? s.tests : [],
      flatten([
        for c in(s.sections != null ? s.sections : []) : concat(
          c.tests != null ? c.tests : [],
          flatten([
            for g in(c.sections != null ? c.sections : []) :
            g.tests != null ? g.tests : []
          ])
        )
      ])
    )
  ])
  high_tests = [for t in local.source_tests : t if t.priority == "High"]
}

resource "orcasecurity_custom_compliance_framework" "subset" {
  name       = "GCP CIS — critical subset"
  visibility = "Organizational"

  sections = [
    {
      name = "Selected controls"
      tests = [
        for t in local.high_tests : { rule_id = t.rule_id }
      ]
    }
  ]
}

resource "orcasecurity_compliance_framework_selection" "subset" {
  framework_id = orcasecurity_custom_compliance_framework.subset.id
  scopes       = ["organization"]
}
