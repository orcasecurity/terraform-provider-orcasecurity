# Build a custom framework from CIS Level 1 and Level 2 controls on a built-in
# one. The single-framework data source returns sections[].tests[].rule_id and
# cis_level; omit rule_id_in_framework and the provider derives it as
# <section_id_in_framework>.<1-based index>. Sibling ids must be unique and
# strictly ascending (7 then 8) because the API returns sections sorted by id.
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
  level1_tests = [for t in local.source_tests : t if contains(coalesce(t.cis_level, []), "Level 1")]
  level2_tests = [for t in local.source_tests : t if contains(coalesce(t.cis_level, []), "Level 2")]
}

resource "orcasecurity_custom_compliance_framework" "subset" {
  name       = "GCP CIS subset"
  visibility = "Organizational"

  sections = [
    {
      name                    = "CIS Level 1"
      section_id_in_framework = "7"
      tests = [
        for t in local.level1_tests : { rule_id = t.rule_id }
      ]
    },
    {
      name                    = "CIS Level 2"
      section_id_in_framework = "8"
      tests = [
        for t in local.level2_tests : { rule_id = t.rule_id }
      ]
    }
  ]
}

resource "orcasecurity_compliance_framework_selection" "subset" {
  framework_id = orcasecurity_custom_compliance_framework.subset.id
  scopes       = ["organization"]
}
