---
page_title: "orcasecurity_automation_v2_priority_order Resource - orcasecurity"
subcategory: ""
description: |-
  Owns the top evaluation-order positions of the organization's automations.
---

# orcasecurity_automation_v2_priority_order (Resource)

Bulk, deterministic ownership of the **top N** automation evaluation-order positions (`priority`
1..N), where N is the number of IDs listed in `automation_ids`. Every apply asserts each listed
automation's priority to its list position: the first entry gets priority 1, the second gets
priority 2, and so on.

This is a **singleton** resource: declare at most one instance per provider configuration —
if you declare a second instance, both will assert priorities 1..N over their own lists and
repeatedly overwrite each other's ordering on every apply, causing perpetual plan diffs. Do
not combine it with the `priority` attribute on `orcasecurity_automation_v2` resources for the
same automations — both mechanisms write to the same global ordering and will fight each other,
producing perpetual plan diffs.

## Choosing between this resource and the `priority` attribute

The organization has a single global automation evaluation order, and Orca exposes only one
write path for it (`PUT /api/automations/{id}/priority`, one automation at a time). Both this
resource and the `priority` attribute on `orcasecurity_automation_v2` drive that same path — so
**pick one per automation, never both**:

| Use the `priority` attribute when… | Use this resource when… |
|---|---|
| You want to pin one or a few individual automations to a position, inline with the resource. | You want one place that owns the **relative order of a set** of automations. |
| The automations that set `priority` are spaced out and rarely reordered together. | You reorder the set as a unit and need a **full reorder to converge in a single apply**. |
| Other automations in the org are managed outside Terraform and you don't want to own the top of the list. | You want the listed automations to occupy the **top N positions** deterministically. |

Why the split matters: with the per-automation `priority` attribute, reordering several automations
in one apply is order-dependent (the server renumbers on each individual write), so it can take
more than one apply to converge. This resource asserts the whole list sequentially from a single
owner, so a full reorder always converges in one apply. That determinism is the reason to use it
for bulk ordering.

## Example Usage

```terraform
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
```

## Behavior notes

- **Applies are verified**: after asserting priorities 1..N, Terraform re-reads the actual top-N
  order and fails the apply if the server did not converge to the requested order. The usual
  cause is legacy duplicate (or gapped) priorities, which the priority API cannot separate —
  the error reports the achieved order; contact Orca support to renumber the organization's
  automation priorities, then re-apply.
- **Priority writes are not atomic**: the order is asserted one automation at a time. A failure
  mid-sequence leaves earlier entries moved; the apply fails, and the next apply re-asserts the
  full order from the top.
- **Drift detection**: on every plan, Terraform re-reads the actual top-N automations from the
  API (N = the number of IDs currently in state) and shows drift if any listed automation was
  reordered, displaced from the top N, or deleted outside Terraform. Apply re-asserts the
  configured order. If a listed automation is deleted outside Terraform, the next apply's
  priority-assignment call fails naming that ID; remove it from `automation_ids` in your
  configuration to recover.
- **Reordering**: changing the order of `automation_ids` (or the list contents) plans an update;
  apply reasserts priorities 1..N in the new order.
- **Delete is a no-op**: destroying this resource simply stops managing the ordering. The
  automations are not deleted and keep whatever priority they had at the time of destroy.
- **Import is not supported**: state is trivially rebuilt by declaring `automation_ids` with the
  desired order and running `terraform apply` (or `plan` to preview) — there is nothing to import.
- **Permissions**: writing priority requires a token with the global Rules Create (admin)
  permission; other tokens receive HTTP 403.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `automation_ids` (List of String) Automation IDs in desired evaluation order; the first entry gets priority 1. Automations not listed keep their relative order below the listed ones.

### Read-Only

- `id` (String) The ID of this resource.


