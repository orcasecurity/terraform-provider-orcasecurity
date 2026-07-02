# Monday Template Resource Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `orcasecurity_integration_monday_template` Terraform resource that manages Monday.com ticket templates via Orca's `/api/external_service/config` endpoint.

**Architecture:** Reuse the existing generic config-integration machinery. The api_client layer aliases `ConfigEnvelope[MondayTemplateConfig]` and uses the shared `CreateExternalServiceConfig`/`Get`/`Update`/`Delete` helpers (the modern pattern from `servicenow_itsm_template.go`). The resource layer declares a `cc.Spec[api_client.MondayTemplate]` on top of `config_integration_common.New`, so there is no hand-rolled CRUD.

**Tech Stack:** Go, HashiCorp terraform-plugin-framework, existing `orcasecurity/api_client` + `orcasecurity/config_integration_common` + `orcasecurity/integrations_common` packages.

## Global Constraints

- Service name: `monday` (exact string, matches `ExternalService.MONDAY.value` in the orca backend).
- `config` is a free-form JSON blob — no client-side validation of mapping structure; Orca validates server-side.
- `resource_id` and `board_id` are the only required inputs; `workspace_id`, `group_id`, and the three `*_json` mapping fields are optional.
- `business_units` is create-only (RequiresReplace); it MUST be omitted from PUT bodies.
- Go source must pass `gofmt`, `go vet`, and `staticcheck` (repo runs these in CI — see recent WASP-1406 commits).
- Follow the existing template packages verbatim for style; do not restructure shared code.

---

### Task 1: api_client — Monday template wire layer

**Files:**
- Create: `orcasecurity/api_client/monday_template.go`
- Test: `orcasecurity/api_client/monday_template_test.go`

**Interfaces:**
- Consumes (already exist in package `api_client`): `ConfigEnvelope[C]`, `CreateExternalServiceConfig[C]`, `GetExternalServiceConfig[C]`, `UpdateExternalServiceConfig[C]`, `DeleteExternalServiceConfig`, `BuildUpdateBody[C]`, `RoundTripFunc`, `APIClient{APIEndpoint, APIToken, HTTPClient}`.
- Produces (relied on by Task 2):
  - `const MondayServiceName = "monday"`
  - `type MondayTemplateConfig struct { WorkspaceID, BoardID, GroupID string; Mapping, AlertStatusMapping, TicketStatusMapping json.RawMessage }`
  - `type MondayTemplate = ConfigEnvelope[MondayTemplateConfig]`
  - `func (c *APIClient) CreateMondayTemplate(p MondayTemplate) (*MondayTemplate, error)`
  - `func (c *APIClient) GetMondayTemplate(name string) (*MondayTemplate, error)`
  - `func (c *APIClient) UpdateMondayTemplate(name string, p MondayTemplate) (*MondayTemplate, error)`
  - `func (c *APIClient) DeleteMondayTemplate(name string) error`

- [ ] **Step 1: Write the failing tests**

Create `orcasecurity/api_client/monday_template_test.go`:

```go
package api_client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"
)

func newMondayTestClient(handler func(req *http.Request) *http.Response) *api_client.APIClient {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(handler)}
	return &api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
}

func TestMondayTemplate_CreateSendsServiceNameResourceAndConfig(t *testing.T) {
	var gotBody map[string]interface{}
	apiClient := newMondayTestClient(func(req *http.Request) *http.Response {
		raw, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(raw, &gotBody)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"id":"cfg-1","service_name":"monday","template_name":"t1","resource":"res-1","is_enabled":true,"is_default":false,"config":{"board_id":"b1"}}}`)),
			Request:    req,
		}
	})

	out, err := apiClient.CreateMondayTemplate(api_client.MondayTemplate{
		TemplateName: "t1",
		Resource:     "res-1",
		IsEnabled:    true,
		Config: api_client.MondayTemplateConfig{
			BoardID:     "b1",
			WorkspaceID: "w1",
			Mapping:     json.RawMessage(`{"status_14":{"value":"0"}}`),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "cfg-1" {
		t.Errorf("expected id cfg-1, got %q", out.ID)
	}
	if gotBody["service_name"] != "monday" {
		t.Errorf("expected service_name monday, got %v", gotBody["service_name"])
	}
	if gotBody["resource"] != "res-1" {
		t.Errorf("expected top-level resource res-1, got %v", gotBody["resource"])
	}
	cfg := gotBody["config"].(map[string]interface{})
	if cfg["board_id"] != "b1" || cfg["workspace_id"] != "w1" {
		t.Errorf("unexpected config: %v", cfg)
	}
}

func TestMondayTemplate_UpdateOmitsBusinessUnitsAndKeepsResource(t *testing.T) {
	var gotBody map[string]interface{}
	apiClient := newMondayTestClient(func(req *http.Request) *http.Response {
		raw, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(raw, &gotBody)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"id":"cfg-1","resource":"res-1","config":{"board_id":"b1"}}}`)),
			Request:    req,
		}
	})

	_, err := apiClient.UpdateMondayTemplate("t1", api_client.MondayTemplate{
		Resource:      "res-1",
		IsEnabled:     true,
		IsDefault:     false,
		BusinessUnits: []string{"bu-1"},
		Config:        api_client.MondayTemplateConfig{BoardID: "b1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := gotBody["business_units"]; present {
		t.Errorf("business_units must be omitted from PUT body, got %v", gotBody["business_units"])
	}
	if gotBody["resource"] != "res-1" {
		t.Errorf("expected resource res-1 in PUT body, got %v", gotBody["resource"])
	}
}

func TestMondayTemplate_GetReturnsFirstEntry(t *testing.T) {
	apiClient := newMondayTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":[{"id":"cfg-1","template_name":"t1","resource":"res-1","config":{"board_id":"b1","group_id":"topics"}}]}`)),
			Request:    req,
		}
	})

	out, err := apiClient.GetMondayTemplate("t1")
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || out.ID != "cfg-1" {
		t.Fatalf("expected cfg-1, got %+v", out)
	}
	if out.Config.BoardID != "b1" || out.Config.GroupID != "topics" {
		t.Errorf("unexpected config: %+v", out.Config)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./orcasecurity/api_client/ -run TestMondayTemplate -v`
Expected: FAIL — compile errors (`undefined: api_client.MondayTemplate`, `CreateMondayTemplate`, etc.).

- [ ] **Step 3: Write the implementation**

Create `orcasecurity/api_client/monday_template.go`:

```go
package api_client

import (
	"encoding/json"
)

const MondayServiceName = "monday"

// MondayTemplateConfig mirrors the "config" block of the monday external_service/config
// payload. The mapping fields are kept as json.RawMessage so the provider preserves the
// customer's arbitrary structure verbatim (Orca validates server-side).
type MondayTemplateConfig struct {
	WorkspaceID         string          `json:"workspace_id,omitempty"`
	BoardID             string          `json:"board_id,omitempty"`
	GroupID             string          `json:"group_id,omitempty"`
	Mapping             json.RawMessage `json:"mapping,omitempty"`
	AlertStatusMapping  json.RawMessage `json:"alert_status_mapping,omitempty"`
	TicketStatusMapping json.RawMessage `json:"ticket_status_mapping,omitempty"`
}

// MondayTemplate aliases the shared envelope. The envelope's Resource field carries the linked
// Monday OAuth resource id.
type MondayTemplate = ConfigEnvelope[MondayTemplateConfig]

func (client *APIClient) CreateMondayTemplate(payload MondayTemplate) (*MondayTemplate, error) {
	return CreateExternalServiceConfig[MondayTemplateConfig](client, MondayServiceName, payload)
}

func (client *APIClient) GetMondayTemplate(templateName string) (*MondayTemplate, error) {
	return GetExternalServiceConfig[MondayTemplateConfig](client, MondayServiceName, templateName, nil)
}

func (client *APIClient) UpdateMondayTemplate(templateName string, payload MondayTemplate) (*MondayTemplate, error) {
	// business_units intentionally omitted — Orca rejects BU changes on update
	// ("You can't change business units"); modelled as RequiresReplace on the Terraform side.
	body := BuildUpdateBody(payload, payload.Config, false)
	if payload.Resource != "" {
		body["resource"] = payload.Resource
	}
	return UpdateExternalServiceConfig[MondayTemplateConfig](client, MondayServiceName, templateName, body)
}

func (client *APIClient) DeleteMondayTemplate(templateName string) error {
	return DeleteExternalServiceConfig(client, MondayServiceName, templateName)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./orcasecurity/api_client/ -run TestMondayTemplate -v`
Expected: PASS (all three tests).

- [ ] **Step 5: Format, vet, commit**

```bash
gofmt -w orcasecurity/api_client/monday_template.go orcasecurity/api_client/monday_template_test.go
go vet ./orcasecurity/api_client/
git add orcasecurity/api_client/monday_template.go orcasecurity/api_client/monday_template_test.go
git commit -m "WASP-1406/add Monday template api_client wire layer"
```

---

### Task 2: Resource — `orcasecurity_integration_monday_template`

**Files:**
- Create: `orcasecurity/monday_template/resource.go`
- Modify: `orcasecurity/provider.go` (add `monday_template.NewMondayTemplateResource` to the `Resources()` slice, near `jira_cloud_template.NewJiraCloudTemplateResource`)

**Interfaces:**
- Consumes: everything Produced by Task 1; plus `config_integration_common` (`cc.Spec`, `cc.New`, `cc.State`, `cc.APIObject`, `cc.CommonFieldsWithBU`), `integrations_common` (`common.DecodeJSONField`, `common.EncodeJSONField`, `common.BusinessUnitsToAPI`).
- Produces: `func NewMondayTemplateResource() resource.Resource`.

- [ ] **Step 1: Write the resource implementation**

Create `orcasecurity/monday_template/resource.go`:

```go
package monday_template

import (
	"context"
	"encoding/json"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// state is the Monday template Terraform model. CommonFieldsWithBU carries id / template_name /
// is_enabled / is_default / business_units plus the GetCommon/SetCommon glue the generic spec
// needs. resource_id maps to the envelope's top-level "resource".
type state struct {
	cc.CommonFieldsWithBU
	ResourceID              types.String `tfsdk:"resource_id"`
	WorkspaceID             types.String `tfsdk:"workspace_id"`
	BoardID                 types.String `tfsdk:"board_id"`
	GroupID                 types.String `tfsdk:"group_id"`
	MappingJSON             types.String `tfsdk:"mapping_json"`
	AlertStatusMappingJSON  types.String `tfsdk:"alert_status_mapping_json"`
	TicketStatusMappingJSON types.String `tfsdk:"ticket_status_mapping_json"`
}

func variantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"resource_id": schema.StringAttribute{
			Required:    true,
			Description: "UUID of the Monday resource that carries the credentials (look it up in the Orca UI under Integrations → Monday.com).",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"board_id": schema.StringAttribute{
			Required:    true,
			Description: "Monday board ID Orca opens items in.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"workspace_id": schema.StringAttribute{
			Optional:    true,
			Description: "Monday workspace ID that owns the board.",
		},
		"group_id": schema.StringAttribute{
			Optional:    true,
			Description: "Monday group (section) ID within the board where new items are created.",
		},
		"mapping_json": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded `mapping` object. Each key is a Monday column ID; values are lists of `{ \"orca\": \"<alert_field>\" }`, a `{ \"custom\": \"<literal>\" }` object, a `{ \"value\": \"<literal>\" }` object, or a list of `{ \"value\": { \"id\": ..., \"kind\": ... } }` entries for people columns.",
		},
		"alert_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded `alert_status_mapping` — maps Orca alert statuses to Monday status column values (for example, `{\"snoozed\": \"1\"}`).",
		},
		"ticket_status_mapping_json": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded `ticket_status_mapping` — maps Monday status column values back to Orca alert state changes (for example, `{\"2\": {\"status\": \"dismissed\"}}`).",
		},
		// Override the base business_units attribute: Orca only accepts this value at create time
		// (updates are rejected with "You can't change business units"), so a change forces replace.
		"business_units": schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Optional set of Orca business unit IDs that may use this template. Orca only accepts this value at create time — changes force Terraform to replace the template.",
			PlanModifiers: []planmodifier.Set{
				setplanmodifier.RequiresReplace(),
			},
		},
	}
}

// decodeMappings pulls the three JSON-string fields off the plan into the API config.
func decodeMappings(s *state, cfg *api_client.MondayTemplateConfig, diags *diag.Diagnostics) {
	for _, m := range []struct {
		src   types.String
		field string
		dst   *json.RawMessage
	}{
		{s.MappingJSON, "mapping_json", &cfg.Mapping},
		{s.AlertStatusMappingJSON, "alert_status_mapping_json", &cfg.AlertStatusMapping},
		{s.TicketStatusMappingJSON, "ticket_status_mapping_json", &cfg.TicketStatusMapping},
	} {
		raw, d := common.DecodeJSONField(m.src, m.field)
		diags.Append(d...)
		*m.dst = raw
	}
}

// encodeMappings writes the three JSON config fields from the API response back onto state,
// preserving each field's planned whitespace shape via EncodeJSONField.
func encodeMappings(s *state, cfg *api_client.MondayTemplateConfig, diags *diag.Diagnostics) {
	for _, m := range []struct {
		raw json.RawMessage
		dst *types.String
	}{
		{cfg.Mapping, &s.MappingJSON},
		{cfg.AlertStatusMapping, &s.AlertStatusMappingJSON},
		{cfg.TicketStatusMapping, &s.TicketStatusMappingJSON},
	} {
		encoded, d := common.EncodeJSONField(m.raw, *m.dst)
		diags.Append(d...)
		*m.dst = encoded
	}
}

func NewMondayTemplateResource() resource.Resource {
	return cc.New(cc.Spec[api_client.MondayTemplate]{
		TypeNameSuffix:        "_integration_monday_template",
		UIName:                "Monday template",
		Description:           "Manage a Monday.com template in Orca. Creates an external service config of `service_name = \"monday\"` linked to an existing Monday resource. Holds the board, group, field-mapping, and status-mapping settings used when Orca opens Monday items.",
		SupportsBusinessUnits: true,
		VariantAttributes:     variantAttributes(),
		NewState:              func() cc.State { return &state{} },
		BuildPayload: func(ctx context.Context, st cc.State, diags *diag.Diagnostics) api_client.MondayTemplate {
			s := st.(*state)
			cfg := api_client.MondayTemplateConfig{
				WorkspaceID: s.WorkspaceID.ValueString(),
				BoardID:     s.BoardID.ValueString(),
				GroupID:     s.GroupID.ValueString(),
			}
			decodeMappings(s, &cfg, diags)
			return api_client.MondayTemplate{
				TemplateName:  s.TemplateName.ValueString(),
				Resource:      s.ResourceID.ValueString(),
				IsEnabled:     s.IsEnabled.ValueBool(),
				IsDefault:     s.IsDefault.ValueBool(),
				Config:        cfg,
				BusinessUnits: common.BusinessUnitsToAPI(ctx, s.BusinessUnits, diags),
			}
		},
		Extract: func(o *api_client.MondayTemplate, st cc.State, diags *diag.Diagnostics) cc.APIObject {
			s := st.(*state)
			if o.Resource != "" {
				s.ResourceID = types.StringValue(o.Resource)
			}
			if o.Config.WorkspaceID != "" {
				s.WorkspaceID = types.StringValue(o.Config.WorkspaceID)
			}
			if o.Config.BoardID != "" {
				s.BoardID = types.StringValue(o.Config.BoardID)
			}
			if o.Config.GroupID != "" {
				s.GroupID = types.StringValue(o.Config.GroupID)
			}
			encodeMappings(s, &o.Config, diags)
			return cc.APIObject{
				ID:            o.ID,
				TemplateName:  o.TemplateName,
				IsEnabled:     o.IsEnabled,
				IsDefault:     o.IsDefault,
				BusinessUnits: o.BusinessUnits,
			}
		},
		Create: (*api_client.APIClient).CreateMondayTemplate,
		Get:    (*api_client.APIClient).GetMondayTemplate,
		Update: (*api_client.APIClient).UpdateMondayTemplate,
		Delete: (*api_client.APIClient).DeleteMondayTemplate,
	})
}
```

- [ ] **Step 2: Register the resource in the provider**

In `orcasecurity/provider.go`, add the import (in the existing import block, alphabetical among the `orcasecurity/...` package imports):

```go
	"terraform-provider-orcasecurity/orcasecurity/monday_template"
```

And add to the `Resources()` slice, immediately after the `jira_cloud_template.NewJiraCloudTemplateResource,` line:

```go
		monday_template.NewMondayTemplateResource,
```

- [ ] **Step 3: Build and vet**

Run: `go build ./... && go vet ./orcasecurity/monday_template/ ./orcasecurity/`
Expected: no output (success). Fixes any wiring/type mismatch before proceeding.

- [ ] **Step 4: Sanity-check the provider schema compiles into the server**

Run: `go test ./orcasecurity/ -run TestProvider -v` (if a provider schema test exists; otherwise skip).
Expected: PASS, or "no tests to run" — either is acceptable. A failure here means the schema is malformed.

- [ ] **Step 5: Format and commit**

```bash
gofmt -w orcasecurity/monday_template/resource.go orcasecurity/provider.go
git add orcasecurity/monday_template/resource.go orcasecurity/provider.go
git commit -m "WASP-1406/add orcasecurity_integration_monday_template resource"
```

---

### Task 3: Example, docs template, generated docs

**Files:**
- Create: `examples/resources/orcasecurity_integration_monday_template/resource.tf`
- Create: `templates/resources/integration_monday_template.md.tmpl` (only if the repo uses per-resource templates — check `ls templates/resources/`; if the doc is fully generated from schema + example, skip the tmpl and let tfplugindocs produce it)
- Create/generate: `docs/resources/integration_monday_template.md`

**Interfaces:**
- Consumes: the resource schema Produced by Task 2 (attribute names must match exactly: `template_name`, `resource_id`, `board_id`, `workspace_id`, `group_id`, `mapping_json`, `alert_status_mapping_json`, `ticket_status_mapping_json`, `business_units`, `is_enabled`, `is_default`).

- [ ] **Step 1: Write the example**

Create `examples/resources/orcasecurity_integration_monday_template/resource.tf`:

```hcl
# Template that defines how Orca alerts map to Monday.com board columns and how
# alert/ticket status changes sync between Orca and Monday.
resource "orcasecurity_integration_monday_template" "demo" {
  template_name = "monday-template-name"
  resource_id   = "49f9b117-e21b-4a70-9763-fc3d44d91e6e"
  workspace_id  = "3709069"
  board_id      = "1827821929"
  group_id      = "topics"

  mapping_json = jsonencode({
    long_text_mkn8v2sp = [{ orca = "alert_id" }, { orca = "asset_name" }]
    text               = [{ orca = "asset_category" }, { orca = "source" }]
    numbers_mkn8zhtt   = { custom = "5" }
    person             = [{ value = { id = "66396150", kind = "person" } }]
    status_14          = { value = "0" }
  })

  alert_status_mapping_json  = jsonencode({ snoozed = "1" })
  ticket_status_mapping_json = jsonencode({ "2" = { status = "dismissed" } })

  business_units = [
    "a411f20b-0276-438c-a9d5-938c48a40957",
    "930c1e5e-6b2e-4881-a393-33491e758144",
  ]

  is_enabled = true
  is_default = false
}
```

- [ ] **Step 2: Check the docs generation convention**

Run: `ls templates/resources/ | head; ls docs/resources/ | grep -iE "jira_cloud|servicenow_itsm"`
Expected: shows whether Jira/ServiceNow templates have hand-written `templates/resources/*.tmpl` or purely generated `docs/resources/*.md`. Mirror whichever convention those template resources use.

- [ ] **Step 3: Generate the docs**

Run: `go generate ./...` (or the repo's documented docs command, e.g. `tfplugindocs generate`).
Expected: creates/updates `docs/resources/integration_monday_template.md`. Verify the file exists and lists all attributes with the descriptions from the schema.

- [ ] **Step 4: Verify docs formatting**

Run: `git status --porcelain docs/ examples/ templates/`
Expected: shows the new Monday example + generated doc (+ tmpl if the convention uses one). Open `docs/resources/integration_monday_template.md` and confirm required/optional flags match the schema (resource_id + board_id required, rest optional).

- [ ] **Step 5: Commit**

```bash
git add examples/resources/orcasecurity_integration_monday_template/ docs/resources/integration_monday_template.md templates/resources/integration_monday_template.md.tmpl
git commit -m "WASP-1406/add Monday template example and docs"
```

(Drop the `templates/...` path from `git add` if Step 2 showed the repo does not use per-resource tmpl files.)

---

### Task 4: Full verification gate

**Files:** none (verification only).

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 2: Full test suite for touched packages**

Run: `go test ./orcasecurity/api_client/ ./orcasecurity/monday_template/ ./orcasecurity/`
Expected: PASS (or "no test files" for the resource package if none were added — acceptable, the api_client tests cover the wire layer).

- [ ] **Step 3: Lint gates (match CI)**

Run: `gofmt -l orcasecurity/ | tee /dev/stderr | (! read)` then `go vet ./...` then `staticcheck ./orcasecurity/... 2>/dev/null || true`
Expected: `gofmt -l` prints nothing (no unformatted files); `go vet` clean; `staticcheck` reports no new issues in the Monday files.

- [ ] **Step 4: Confirm resource is registered**

Run: `grep -n "NewMondayTemplateResource" orcasecurity/provider.go`
Expected: exactly one match inside the `Resources()` slice.

- [ ] **Step 5: Final commit (only if the lint gate produced formatting fixes)**

```bash
git add -A
git commit -m "WASP-1406/gofmt and lint fixes for Monday template" || echo "nothing to commit"
```

---

## Self-Review

**Spec coverage:**
- api_client wire layer (ConfigEnvelope + generic helpers, service_name=monday, top-level resource, PUT omits business_units) → Task 1. ✅
- Resource on `cc.Spec` with the exact attribute table, resource_id+board_id required, RequiresReplace business_units, mapping helper tables → Task 2. ✅
- Provider registration → Task 2 Step 2. ✅
- api_client unit tests → Task 1. ✅
- Example + docs → Task 3. ✅
- No data source → nothing added; confirmed out of scope. ✅

**Placeholder scan:** All code blocks are complete; no TBD/TODO. Task 3 tmpl file is conditional with an explicit check step, not a placeholder. ✅

**Type consistency:** `MondayTemplateConfig` field names (`WorkspaceID`, `BoardID`, `GroupID`, `Mapping`, `AlertStatusMapping`, `TicketStatusMapping`) used identically in Task 1 (definition + tests) and Task 2 (BuildPayload/Extract/decode/encode). TF attribute names (`resource_id`, `board_id`, `workspace_id`, `group_id`, `mapping_json`, `alert_status_mapping_json`, `ticket_status_mapping_json`) match between schema, state struct tfsdk tags, and the example. CRUD method names (`CreateMondayTemplate`/`GetMondayTemplate`/`UpdateMondayTemplate`/`DeleteMondayTemplate`) consistent across Tasks 1 and 2. ✅
