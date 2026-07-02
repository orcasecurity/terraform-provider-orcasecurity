# Monday Template Resource — Design

**Date:** 2026-07-02
**Status:** Approved
**Scope:** Add `orcasecurity_integration_monday_template` resource. No data source.

## Goal

Let Terraform manage Monday.com ticket templates in Orca, the same way
`orcasecurity_integration_jira_cloud_template` and the ServiceNow templates manage their
respective `/api/external_service/config` entries. A Monday template pins the board, group,
field-mapping, and status-mapping settings Orca uses when it opens Monday items for alerts.

## Backend reference (confirmed in `orca` repo)

- Endpoint: `POST/GET/PUT/DELETE /api/external_service/config`, `service_name = "monday"`.
- `config` is a free-form `JSONField` — no strict per-field validation server-side.
- **`board_id`** — required at runtime (`config["board_id"]` accessed directly in
  `monday/status_updater.py` and `monday/use_cases.py`).
- **`resource`** (top-level) — required for ticket creation; the Monday client is built from
  `external_service_config.resource`.
- `workspace_id`, `group_id`, `mapping`, `alert_status_mapping`, `ticket_status_mapping` —
  all optional server-side (`.get()` with fallbacks; backend tests use `config={"board_id":"1"}`).
- `business_units` — treated as create-only (same guard as Jira: "You can't change business
  units"), so it is omitted from PUT bodies and modelled as RequiresReplace.

## Example payload (from user)

```json
{
  "is_default": false,
  "is_enabled": true,
  "template_name": "Monday_Template_name",
  "service_name": "monday",
  "business_units": ["a411f20b-...", "930c1e5e-..."],
  "resource": "49f9b117-...",
  "config": {
    "workspace_id": "3709069",
    "board_id": "1827821929",
    "group_id": "topics",
    "mapping": { "long_text_mkn8v2sp": [{"orca": "alert_id"}], "status_14": {"value": "0"} },
    "alert_status_mapping": {"snoozed": "1"},
    "ticket_status_mapping": {"2": {"status": "dismissed"}}
  }
}
```

## Architecture

Reuse the existing generic machinery — no new CRUD loop:

- **`config_integration_common.Spec[P]`** drives Create/Read/Update/Delete/Import.
- **`api_client.ConfigEnvelope[C]`** + `CreateExternalServiceConfig` / `GetExternalServiceConfig`
  / `UpdateExternalServiceConfig` / `DeleteExternalServiceConfig` / `BuildUpdateBody` handle the
  wire layer (the modern pattern used by `servicenow_itsm_template.go`, not the older
  hand-rolled `jira_cloud_template.go`).
- **`integrations_common.DecodeJSONField` / `EncodeJSONField`** round-trip the free-form JSON
  mapping fields so plans do not drift on whitespace.

### New file: `orcasecurity/api_client/monday_template.go`

```go
const MondayServiceName = "monday"

type MondayTemplateConfig struct {
    WorkspaceID         string          `json:"workspace_id,omitempty"`
    BoardID             string          `json:"board_id,omitempty"`
    GroupID             string          `json:"group_id,omitempty"`
    Mapping             json.RawMessage `json:"mapping,omitempty"`
    AlertStatusMapping  json.RawMessage `json:"alert_status_mapping,omitempty"`
    TicketStatusMapping json.RawMessage `json:"ticket_status_mapping,omitempty"`
}

type MondayTemplate = ConfigEnvelope[MondayTemplateConfig]

func (c *APIClient) CreateMondayTemplate(p MondayTemplate) (*MondayTemplate, error) {
    return CreateExternalServiceConfig[MondayTemplateConfig](c, MondayServiceName, p)
}
func (c *APIClient) GetMondayTemplate(name string) (*MondayTemplate, error) {
    return GetExternalServiceConfig[MondayTemplateConfig](c, MondayServiceName, name, nil)
}
func (c *APIClient) UpdateMondayTemplate(name string, p MondayTemplate) (*MondayTemplate, error) {
    // business_units omitted (RequiresReplace); re-add top-level resource like ServiceNow.
    body := BuildUpdateBody(p, p.Config, false)
    if p.Resource != "" {
        body["resource"] = p.Resource
    }
    return UpdateExternalServiceConfig[MondayTemplateConfig](c, MondayServiceName, name, body)
}
func (c *APIClient) DeleteMondayTemplate(name string) error {
    return DeleteExternalServiceConfig(c, MondayServiceName, name)
}
```

### New file: `orcasecurity/monday_template/resource.go`

State model on `cc.CommonFieldsWithBU` (id / template_name / is_enabled / is_default /
business_units) plus:

| TF attribute | maps to | required |
|---|---|---|
| `resource_id` | envelope `Resource` (top-level) | **required** |
| `board_id` | `config.board_id` | **required** |
| `workspace_id` | `config.workspace_id` | optional |
| `group_id` | `config.group_id` | optional |
| `mapping_json` | `config.mapping` | optional |
| `alert_status_mapping_json` | `config.alert_status_mapping` | optional |
| `ticket_status_mapping_json` | `config.ticket_status_mapping` | optional |
| `business_units` | envelope `business_units` | optional, RequiresReplace |

- `resource_id` and `board_id` use `stringvalidator.LengthAtLeast(1)`.
- `business_units` overridden with a `setplanmodifier.RequiresReplace()` plan modifier and the
  "changes force replace" description — identical to `jira_cloud_template`.
- Mapping fields decoded/encoded through a small helper table (same shape as
  `jira_cloud_template.decodeMappings` / `encodeMappings`), using `common.DecodeJSONField` /
  `EncodeJSONField`.
- `Spec` wiring: `TypeNameSuffix: "_integration_monday_template"`,
  `UIName: "Monday template"`, `SupportsBusinessUnits: true`, CRUD refs pointing at the four
  `api_client` methods above. `BuildPayload` sets `Resource` from `resource_id`. `Extract`
  copies non-empty `Resource`, `board_id`, `workspace_id`, `group_id` back to state and
  re-encodes the three mapping fields.

### Registration

Add `monday_template.NewMondayTemplateResource` to the `Resources()` slice in
`orcasecurity/provider.go`.

## Testing

- `orcasecurity/api_client/monday_template_test.go` — unit tests mirroring an existing template
  api_client test: Create/Get/Update/Delete round-trip against an `httptest` server, asserting
  `service_name=monday`, top-level `resource`, config field names, and that PUT omits
  `business_units`.
- Provider acceptance tests follow the existing template pattern only if the repo already runs
  them for Jira/ServiceNow templates; otherwise unit coverage at the api_client layer plus a
  `go build` / `go vet` gate.

## Docs & examples

- `examples/resources/orcasecurity_integration_monday_template/resource.tf` — worked example
  built from the user's payload (board/group/workspace ids + a representative mapping).
- `templates/resources/` doc template + generated `docs/resources/` page (via `tfplugindocs`,
  matching how the other template docs are produced).

## Out of scope

- No data source (workspaces/boards/groups/columns lookup). Addable later without breaking the
  resource.
- No client-side validation of mapping structure — Orca validates server-side.
