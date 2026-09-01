## 0.1.0 (Unreleased)

FEATURES:

* **New Resource:** `orcasecurity_compliance_framework_selection` — enable/disable a compliance framework per `user`/`organization` scope. `scopes = []` is the explicit disable action. Destroy is state-only by default (built-in frameworks are not Terraform's to switch off); `restore_on_destroy` opts back in.
* **New Data Source:** `orcasecurity_compliance_frameworks` — list/filter the tenant's select map.
* **New Data Source:** `orcasecurity_compliance_framework` — one framework plus its catalog section tree.

ENHANCEMENTS:

* `orcasecurity_custom_compliance_framework`: nested sections (three levels), optional `description`/`tests`, `visibility`, `forced_cloud_vendors`, create-only `scope`, and extra test fields (`priority`, `control_unique_id`, `origin_framework_id`, `reference_id`). `rule_id_in_framework` is now optional and derived as `<section-id>.<1-based index>` when omitted.

BUG FIXES:

* `orcasecurity_custom_compliance_framework`: Read now populates sections from `GET /api/compliance/catalog/{id}` (import no longer lands with empty sections). `GetCustomComplianceFramework` treats only HTTP 404 as gone; 400/500 no longer silently drop the resource from state.

NOTES:

* Existing `orcasecurity_custom_compliance_framework` resources may show a first-plan sections diff after upgrade if state drifted from the live catalog. That is the Read bug being fixed.
