---
page_title: "orcasecurity_shift_left_policy Resource - orcasecurity"
subcategory: ""
description: |-
  Provides an AppSec (Shift Left) policy resource.
---

# orcasecurity_shift_left_policy (Resource)

Provides an AppSec (Shift Left) policy resource. Use this resource to create, update, and delete AppSec scan policies in Orca Security, and to import existing ones.

Supported policy `type` values: `iac`, `sast`, `file_system`, `file_system_vulnerabilities`, `file_system_secret_detection`, `container_image`, `scm_posture`, `licenses`, `sca`.

## Example Usage

### Reference a catalog control by title

You do not need the `orcasecurity_shift_left_policy_catalog_controls` data source to look up IDs. Omit `id` and set `title`; the provider resolves the catalog control ID automatically.

```terraform
resource "orcasecurity_shift_left_policy" "iac_baseline" {
  type                       = "iac"
  name                       = "IaC baseline"
  description                = "Managed by Terraform"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"

  iac {
    controls {
      title    = "API Gateway is publicly accessible"
      priority = "HIGH"
      disabled = false
    }
  }
}
```

### Reference a catalog control by ID

```terraform
resource "orcasecurity_shift_left_policy" "iac_by_id" {
  type                       = "iac"
  name                       = "IaC by id"
  priority_failure_threshold = "HIGH"

  iac {
    controls {
      id       = "f771bf06-ecda-4925-ba7b-08508875c0a6"
      priority = "HIGH"
      disabled = false
    }
  }
}
```

### Include all catalog controls for a section

Set `all_controls = true` on a section to automatically include every catalog control for it (no data source, no enumerating IDs). Available on the `iac`, `sast`, `file_system*`, `licenses`, `sca` blocks and on each `container_image` feature-scope block.

```terraform
resource "orcasecurity_shift_left_policy" "container_all" {
  type                       = "container_image"
  name                       = "Container all controls"
  priority_failure_threshold = "HIGH"

  container_image {
    feature_scope = ["vulnerabilities", "secret_detection", "container_image_best_practices"]

    vulnerabilities {
      all_controls = true
    }
    secret_detection {
      all_controls = true
    }
    container_image_best_practices {
      all_controls = true
    }
  }
}
```

### Create a brand-new custom control

Omit `id` and use a `title` that does not match any catalog control to define a custom control entirely from its `priority` and `conditions`.

```terraform
resource "orcasecurity_shift_left_policy" "container_custom" {
  type                       = "container_image"
  name                       = "Container custom vuln"
  priority_failure_threshold = "HIGH"

  container_image {
    feature_scope = ["vulnerabilities"]

    vulnerabilities {
      controls {
        title    = "My custom vuln control"
        priority = "CRITICAL"
        disabled = false

        conditions {
          severities_operator = "IN"
          severities_values   = ["CRITICAL", "HIGH"]
          fix_available       = true
          has_exploit         = true
          days_from_discovery = 14
        }
      }
    }
  }
}
```

### Attach a policy to projects

```terraform
resource "orcasecurity_shift_left_policy" "attached" {
  type                       = "iac"
  name                       = "Attached policy"
  priority_failure_threshold = "HIGH"

  projects_ids = [orcasecurity_shift_left_project.example.id]

  iac {
    all_controls = true
  }
}
```

### Attach a fleet of projects to a built-in policy

Built-in Orca policies (for example the built-in "OSS Licenses Policy") cannot be
created, renamed, or otherwise edited via Terraform, but -- just like in the Orca
UI -- you can attach projects to one. Import the built-in policy first, then
apply a configuration that changes only `projects_ids`; every other attribute
must match the policy's current values exactly or the apply is rejected.

```shell
terraform import orcasecurity_shift_left_policy.licenses_builtin licenses/<policy-id>
```

```terraform
resource "orcasecurity_shift_left_project" "fleet" {
  for_each         = toset(["team-a", "team-b", "team-c"])
  name             = each.value
  key              = each.value
  default_policies = false
}

resource "orcasecurity_shift_left_policy" "licenses_builtin" {
  type                       = "licenses"
  name                       = "OSS Licenses Policy"
  description                = "Orca built-in open-source license compliance policy (Fail on disallowed or high-risk open source licenses)."
  disabled                   = false
  warn_mode                  = true
  priority_failure_threshold = "HIGH"

  # Bulk-attach the whole fleet by projecting ids out of a for_each collection.
  projects_ids = [for p in orcasecurity_shift_left_project.fleet : p.id]

  licenses {}
}
```

## Update and delete

- **Update**: change any attribute (for example `name`, `description`, `warn_mode`, `priority_failure_threshold`, `disabled`, `projects_ids`, or the controls) and re-apply.
- **Delete**: remove the resource from configuration (or run `terraform destroy`) and apply.
- **Built-in policies**: only `projects_ids` can be changed via Terraform (mirroring what the Orca UI allows); changing any other attribute is rejected, and built-in policies can never be deleted via Terraform.

## Import

Existing policies can be imported. The import ID is `<type>/<policy-id>`, where `<policy-id>` is the UUID from the policy URL in the Orca console (e.g. `.../shift-left/policy/container_image/<policy-id>`).

```shell
# container_image policy
terraform import orcasecurity_shift_left_policy.example container_image/9472f518-10dd-469e-b1fc-b1e451b87584

# iac policy
terraform import orcasecurity_shift_left_policy.example iac/<policy-id>
```

After importing, run `terraform plan` and copy the populated control blocks into your configuration so the plan is empty.

~> **Note:** Built-in Orca policies can only have `projects_ids` changed via Terraform; every other attribute is read-only for built-ins and any other change is rejected. Built-in policies can never be deleted via Terraform. Because built-ins already exist outside of Terraform, always `terraform import` a built-in before changing its `projects_ids` -- Create would otherwise attempt to POST a brand new policy.

-> **Note:** Custom controls are identified by their `title`. Keep custom control titles unique within a section, and do not reuse a catalog control title for a custom control (a matching title is resolved to the catalog control).

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `disabled` (Boolean) Whether the policy is disabled.
- `name` (String) Policy name.
- `priority_failure_threshold` (String) Minimum control priority that causes scan failure.
- `type` (String) Policy type.
- `warn_mode` (Boolean) When true, policy violations produce warnings instead of failures.

### Optional

- `container_image` (Block, Optional) (see [below for nested schema](#nestedblock--container_image))
- `description` (String) Policy description.
- `file_system` (Block, Optional) (see [below for nested schema](#nestedblock--file_system))
- `file_system_secret_detection` (Block, Optional) (see [below for nested schema](#nestedblock--file_system_secret_detection))
- `file_system_vulnerabilities` (Block, Optional) (see [below for nested schema](#nestedblock--file_system_vulnerabilities))
- `iac` (Block, Optional) (see [below for nested schema](#nestedblock--iac))
- `licenses` (Block, Optional) (see [below for nested schema](#nestedblock--licenses))
- `projects_ids` (List of String) Project IDs to attach this policy to.
- `sast` (Block, Optional) (see [below for nested schema](#nestedblock--sast))
- `sca` (Block, Optional) (see [below for nested schema](#nestedblock--sca))
- `scm_posture` (Block, Optional) (see [below for nested schema](#nestedblock--scm_posture))

### Read-Only

- `builtin` (Boolean) Whether this is an Orca built-in policy. Built-in policies cannot be updated or deleted via Terraform.
- `id` (String) AppSec policy ID.

<a id="nestedblock--container_image"></a>
### Nested Schema for `container_image`

Optional:

- `container_image_best_practices` (Block, Optional) (see [below for nested schema](#nestedblock--container_image--container_image_best_practices))
- `custom` (Block, Optional) (see [below for nested schema](#nestedblock--container_image--custom))
- `feature_scope` (List of String) Enabled feature scopes: vulnerabilities, secret_detection, container_image_best_practices, custom.
- `secret_detection` (Block, Optional) (see [below for nested schema](#nestedblock--container_image--secret_detection))
- `vulnerabilities` (Block, Optional) (see [below for nested schema](#nestedblock--container_image--vulnerabilities))

<a id="nestedblock--container_image--container_image_best_practices"></a>
### Nested Schema for `container_image.container_image_best_practices`

Optional:

- `all_controls` (Boolean) When true, include every catalog control for this section automatically (no need to list controls or use a data source).
- `controls` (Block List) (see [below for nested schema](#nestedblock--container_image--container_image_best_practices--controls))

<a id="nestedblock--container_image--container_image_best_practices--controls"></a>
### Nested Schema for `container_image.container_image_best_practices.controls`

Required:

- `disabled` (Boolean)
- `priority` (String)

Optional:

- `conditions` (Block, Optional) (see [below for nested schema](#nestedblock--container_image--container_image_best_practices--controls--conditions))
- `id` (String) Catalog control ID. Omit to define a custom control identified by its title and conditions.
- `origin` (String)
- `title` (String) Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.

<a id="nestedblock--container_image--container_image_best_practices--controls--conditions"></a>
### Nested Schema for `container_image.container_image_best_practices.controls.conditions`

Optional:

- `days_from_discovery` (Number)
- `days_from_fix` (Number)
- `fix_available` (Boolean)
- `from_base_image` (Boolean)
- `has_exploit` (Boolean)
- `severities_operator` (String) Severity filter operator (e.g. IN, NOT_IN).
- `severities_values` (List of String) Severity values for the filter (e.g. CRITICAL, HIGH).




<a id="nestedblock--container_image--custom"></a>
### Nested Schema for `container_image.custom`

Optional:

- `all_controls` (Boolean) When true, include every catalog control for this section automatically (no need to list controls or use a data source).
- `controls` (Block List) (see [below for nested schema](#nestedblock--container_image--custom--controls))

<a id="nestedblock--container_image--custom--controls"></a>
### Nested Schema for `container_image.custom.controls`

Required:

- `disabled` (Boolean)
- `priority` (String)

Optional:

- `conditions` (Block, Optional) (see [below for nested schema](#nestedblock--container_image--custom--controls--conditions))
- `id` (String) Catalog control ID. Omit to define a custom control identified by its title and conditions.
- `origin` (String)
- `title` (String) Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.

<a id="nestedblock--container_image--custom--controls--conditions"></a>
### Nested Schema for `container_image.custom.controls.conditions`

Optional:

- `days_from_discovery` (Number)
- `days_from_fix` (Number)
- `fix_available` (Boolean)
- `from_base_image` (Boolean)
- `has_exploit` (Boolean)
- `severities_operator` (String) Severity filter operator (e.g. IN, NOT_IN).
- `severities_values` (List of String) Severity values for the filter (e.g. CRITICAL, HIGH).




<a id="nestedblock--container_image--secret_detection"></a>
### Nested Schema for `container_image.secret_detection`

Optional:

- `all_controls` (Boolean) When true, include every catalog control for this section automatically (no need to list controls or use a data source).
- `controls` (Block List) (see [below for nested schema](#nestedblock--container_image--secret_detection--controls))

<a id="nestedblock--container_image--secret_detection--controls"></a>
### Nested Schema for `container_image.secret_detection.controls`

Required:

- `disabled` (Boolean)
- `priority` (String)

Optional:

- `conditions` (Block, Optional) (see [below for nested schema](#nestedblock--container_image--secret_detection--controls--conditions))
- `id` (String) Catalog control ID. Omit to define a custom control identified by its title and conditions.
- `origin` (String)
- `title` (String) Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.

<a id="nestedblock--container_image--secret_detection--controls--conditions"></a>
### Nested Schema for `container_image.secret_detection.controls.conditions`

Optional:

- `days_from_discovery` (Number)
- `days_from_fix` (Number)
- `fix_available` (Boolean)
- `from_base_image` (Boolean)
- `has_exploit` (Boolean)
- `severities_operator` (String) Severity filter operator (e.g. IN, NOT_IN).
- `severities_values` (List of String) Severity values for the filter (e.g. CRITICAL, HIGH).




<a id="nestedblock--container_image--vulnerabilities"></a>
### Nested Schema for `container_image.vulnerabilities`

Optional:

- `all_controls` (Boolean) When true, include every catalog control for this section automatically (no need to list controls or use a data source).
- `controls` (Block List) (see [below for nested schema](#nestedblock--container_image--vulnerabilities--controls))

<a id="nestedblock--container_image--vulnerabilities--controls"></a>
### Nested Schema for `container_image.vulnerabilities.controls`

Required:

- `disabled` (Boolean)
- `priority` (String)

Optional:

- `conditions` (Block, Optional) (see [below for nested schema](#nestedblock--container_image--vulnerabilities--controls--conditions))
- `id` (String) Catalog control ID. Omit to define a custom control identified by its title and conditions.
- `origin` (String)
- `title` (String) Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.

<a id="nestedblock--container_image--vulnerabilities--controls--conditions"></a>
### Nested Schema for `container_image.vulnerabilities.controls.conditions`

Optional:

- `days_from_discovery` (Number)
- `days_from_fix` (Number)
- `fix_available` (Boolean)
- `from_base_image` (Boolean)
- `has_exploit` (Boolean)
- `severities_operator` (String) Severity filter operator (e.g. IN, NOT_IN).
- `severities_values` (List of String) Severity values for the filter (e.g. CRITICAL, HIGH).





<a id="nestedblock--file_system"></a>
### Nested Schema for `file_system`

Optional:

- `all_controls` (Boolean) When true, include every catalog control for this section automatically (no need to list controls or use a data source).
- `controls` (Block List) (see [below for nested schema](#nestedblock--file_system--controls))

<a id="nestedblock--file_system--controls"></a>
### Nested Schema for `file_system.controls`

Required:

- `disabled` (Boolean)
- `priority` (String)

Optional:

- `conditions` (Block, Optional) (see [below for nested schema](#nestedblock--file_system--controls--conditions))
- `id` (String) Catalog control ID. Omit to define a custom control identified by its title and conditions.
- `title` (String) Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.

<a id="nestedblock--file_system--controls--conditions"></a>
### Nested Schema for `file_system.controls.conditions`

Optional:

- `days_from_discovery` (Number)
- `days_from_fix` (Number)
- `fix_available` (Boolean)
- `from_base_image` (Boolean)
- `has_exploit` (Boolean)
- `severities_operator` (String) Severity filter operator (e.g. IN, NOT_IN).
- `severities_values` (List of String) Severity values for the filter (e.g. CRITICAL, HIGH).




<a id="nestedblock--file_system_secret_detection"></a>
### Nested Schema for `file_system_secret_detection`

Optional:

- `all_controls` (Boolean) When true, include every catalog control for this section automatically (no need to list controls or use a data source).
- `controls` (Block List) (see [below for nested schema](#nestedblock--file_system_secret_detection--controls))

<a id="nestedblock--file_system_secret_detection--controls"></a>
### Nested Schema for `file_system_secret_detection.controls`

Required:

- `disabled` (Boolean)
- `priority` (String)

Optional:

- `conditions` (Block, Optional) (see [below for nested schema](#nestedblock--file_system_secret_detection--controls--conditions))
- `id` (String) Catalog control ID. Omit to define a custom control identified by its title and conditions.
- `title` (String) Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.

<a id="nestedblock--file_system_secret_detection--controls--conditions"></a>
### Nested Schema for `file_system_secret_detection.controls.conditions`

Optional:

- `days_from_discovery` (Number)
- `days_from_fix` (Number)
- `fix_available` (Boolean)
- `from_base_image` (Boolean)
- `has_exploit` (Boolean)
- `severities_operator` (String) Severity filter operator (e.g. IN, NOT_IN).
- `severities_values` (List of String) Severity values for the filter (e.g. CRITICAL, HIGH).




<a id="nestedblock--file_system_vulnerabilities"></a>
### Nested Schema for `file_system_vulnerabilities`

Optional:

- `all_controls` (Boolean) When true, include every catalog control for this section automatically (no need to list controls or use a data source).
- `controls` (Block List) (see [below for nested schema](#nestedblock--file_system_vulnerabilities--controls))

<a id="nestedblock--file_system_vulnerabilities--controls"></a>
### Nested Schema for `file_system_vulnerabilities.controls`

Required:

- `disabled` (Boolean)
- `priority` (String)

Optional:

- `conditions` (Block, Optional) (see [below for nested schema](#nestedblock--file_system_vulnerabilities--controls--conditions))
- `id` (String) Catalog control ID. Omit to define a custom control identified by its title and conditions.
- `title` (String) Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.

<a id="nestedblock--file_system_vulnerabilities--controls--conditions"></a>
### Nested Schema for `file_system_vulnerabilities.controls.conditions`

Optional:

- `days_from_discovery` (Number)
- `days_from_fix` (Number)
- `fix_available` (Boolean)
- `from_base_image` (Boolean)
- `has_exploit` (Boolean)
- `severities_operator` (String) Severity filter operator (e.g. IN, NOT_IN).
- `severities_values` (List of String) Severity values for the filter (e.g. CRITICAL, HIGH).




<a id="nestedblock--iac"></a>
### Nested Schema for `iac`

Optional:

- `all_controls` (Boolean) When true, include every catalog control for this section automatically (no need to list controls or use a data source).
- `controls` (Block List) (see [below for nested schema](#nestedblock--iac--controls))

<a id="nestedblock--iac--controls"></a>
### Nested Schema for `iac.controls`

Required:

- `disabled` (Boolean)
- `priority` (String)

Optional:

- `conditions` (Block, Optional) (see [below for nested schema](#nestedblock--iac--controls--conditions))
- `frameworks` (List of String)
- `id` (String) Catalog control ID. Omit to define a custom control identified by its title and conditions.
- `orca_alert_rule_type` (String)
- `title` (String) Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.

<a id="nestedblock--iac--controls--conditions"></a>
### Nested Schema for `iac.controls.conditions`

Optional:

- `days_from_discovery` (Number)
- `days_from_fix` (Number)
- `fix_available` (Boolean)
- `from_base_image` (Boolean)
- `has_exploit` (Boolean)
- `severities_operator` (String) Severity filter operator (e.g. IN, NOT_IN).
- `severities_values` (List of String) Severity values for the filter (e.g. CRITICAL, HIGH).




<a id="nestedblock--licenses"></a>
### Nested Schema for `licenses`

Optional:

- `all_controls` (Boolean) When true, include every catalog control for this section automatically (no need to list controls or use a data source).
- `controls` (Block List) (see [below for nested schema](#nestedblock--licenses--controls))

<a id="nestedblock--licenses--controls"></a>
### Nested Schema for `licenses.controls`

Required:

- `disabled` (Boolean)
- `priority` (String)

Optional:

- `additional_info` (List of String)
- `conditions` (Block, Optional) (see [below for nested schema](#nestedblock--licenses--controls--conditions))
- `id` (String) Catalog control ID. Omit to define a custom control identified by its title and conditions.
- `is_deprecated` (Boolean)
- `is_fsf_libre` (Boolean)
- `is_osi_approved` (Boolean)
- `license_category` (String)
- `license_id` (String)
- `title` (String) Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.
- `url` (String)

<a id="nestedblock--licenses--controls--conditions"></a>
### Nested Schema for `licenses.controls.conditions`

Optional:

- `days_from_discovery` (Number)
- `days_from_fix` (Number)
- `fix_available` (Boolean)
- `from_base_image` (Boolean)
- `has_exploit` (Boolean)
- `severities_operator` (String) Severity filter operator (e.g. IN, NOT_IN).
- `severities_values` (List of String) Severity values for the filter (e.g. CRITICAL, HIGH).




<a id="nestedblock--sast"></a>
### Nested Schema for `sast`

Optional:

- `all_controls` (Boolean) When true, include every catalog control for this section automatically (no need to list controls or use a data source).
- `controls` (Block List) (see [below for nested schema](#nestedblock--sast--controls))

<a id="nestedblock--sast--controls"></a>
### Nested Schema for `sast.controls`

Required:

- `disabled` (Boolean)
- `priority` (String)

Optional:

- `conditions` (Block, Optional) (see [below for nested schema](#nestedblock--sast--controls--conditions))
- `confidence` (String)
- `cwe` (List of String)
- `id` (String) Catalog control ID. Omit to define a custom control identified by its title and conditions.
- `impact` (String)
- `languages` (List of String)
- `likelihood` (String)
- `owasp` (List of String)
- `section` (String)
- `title` (String) Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.

<a id="nestedblock--sast--controls--conditions"></a>
### Nested Schema for `sast.controls.conditions`

Optional:

- `days_from_discovery` (Number)
- `days_from_fix` (Number)
- `fix_available` (Boolean)
- `from_base_image` (Boolean)
- `has_exploit` (Boolean)
- `severities_operator` (String) Severity filter operator (e.g. IN, NOT_IN).
- `severities_values` (List of String) Severity values for the filter (e.g. CRITICAL, HIGH).




<a id="nestedblock--sca"></a>
### Nested Schema for `sca`

Optional:

- `all_controls` (Boolean) When true, include every catalog control for this section automatically (no need to list controls or use a data source).
- `controls` (Block List) (see [below for nested schema](#nestedblock--sca--controls))

<a id="nestedblock--sca--controls"></a>
### Nested Schema for `sca.controls`

Required:

- `disabled` (Boolean)
- `priority` (String)

Optional:

- `additional_info` (List of String)
- `conditions` (Block, Optional) (see [below for nested schema](#nestedblock--sca--controls--conditions))
- `id` (String) Catalog control ID. Omit to define a custom control identified by its title and conditions.
- `is_deprecated` (Boolean)
- `is_fsf_libre` (Boolean)
- `is_osi_approved` (Boolean)
- `license_category` (String)
- `license_id` (String)
- `title` (String) Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.
- `url` (String)

<a id="nestedblock--sca--controls--conditions"></a>
### Nested Schema for `sca.controls.conditions`

Optional:

- `days_from_discovery` (Number)
- `days_from_fix` (Number)
- `fix_available` (Boolean)
- `from_base_image` (Boolean)
- `has_exploit` (Boolean)
- `severities_operator` (String) Severity filter operator (e.g. IN, NOT_IN).
- `severities_values` (List of String) Severity values for the filter (e.g. CRITICAL, HIGH).




<a id="nestedblock--scm_posture"></a>
### Nested Schema for `scm_posture`

Optional:

- `controls` (Block List) (see [below for nested schema](#nestedblock--scm_posture--controls))
- `scope` (Block List) (see [below for nested schema](#nestedblock--scm_posture--scope))

<a id="nestedblock--scm_posture--controls"></a>
### Nested Schema for `scm_posture.controls`

Required:

- `disabled` (Boolean)
- `id` (String)
- `priority` (String)

Optional:

- `entity` (String)
- `scm` (String)
- `threat` (List of String)


<a id="nestedblock--scm_posture--scope"></a>
### Nested Schema for `scm_posture.scope`

Required:

- `ids` (List of String)
- `key` (String) Scope key such as github_installations, github_repository_installations, gitlab_groups, gitlab_repositories.




