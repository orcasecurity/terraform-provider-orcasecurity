---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "orcasecurity_group Resource - orcasecurity"
subcategory: ""
description: |-
  Provides a group resource.
---

# orcasecurity_group (Resource)

Provides a group resource.

## Example Usage

```terraform
//Group
resource "orcasecurity_group" "tf-group-1" {
  name = "Orca Terraform Group 1"

  sso_group   = true
  description = "string"
  users = [
    "{place-user-id-here}"
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `description` (String) Group description.
- `name` (String) Group name. Must be unique across your Orca org.
- `sso_group` (Boolean) Configures whether this group may be used for SSO permissions, or if it should be used purely for use within Orca.
- `users` (Set of String) Users within the group, identified by their IDs. IDs can be determined from the /api/users endpoint.

### Read-Only

- `id` (String) Group ID.
