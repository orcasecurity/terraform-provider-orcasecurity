// Package config_integration_common holds the resource skeleton shared by every
// /api/external_service/config integration (PagerDuty, Opsgenie, Snyk, Splunk, Cloudflare,
// Akamai, Zscaler ZPA, Terraform Cloud, Azure Sentinel). Each per-variant file declares the
// state struct (with tfsdk tags), a Spec carrying the variant-specific attributes /
// payload-encoding / api_client method references, and calls New() — no per-variant CRUD or
// resource-type boilerplate.
package config_integration_common

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Common carries the cross-variant fields every config-endpoint integration declares.
type Common struct {
	ID            types.String
	TemplateName  types.String
	IsEnabled     types.Bool
	IsDefault     types.Bool
	BusinessUnits types.Set
}

// APIObject is the API-shape view of those fields.
type APIObject struct {
	ID            string
	TemplateName  string
	IsEnabled     bool
	IsDefault     bool
	BusinessUnits []string
}

// State is the per-variant TF model. Implementations expose the cross-variant fields via
// Get/Set so the base can refresh them after a CRUD call without knowing the concrete type.
type State interface {
	GetCommon() *Common
	SetCommon(Common)
}

// Spec is parameterised over the variant's api_client payload type. Per-variant code provides
// the schema attributes, payload encoding, response extraction, and api_client method refs;
// everything else lives in this package.
type Spec[P any] struct {
	TypeNameSuffix        string
	UIName                string
	Description           string
	SupportsBusinessUnits bool
	VariantAttributes     map[string]schema.Attribute

	// NewState constructs an empty plan/state model the framework can decode into.
	NewState func() State

	// BuildPayload converts the planned state into an API payload.
	BuildPayload func(ctx context.Context, state State, diags *diag.Diagnostics) P

	// Extract pulls the cross-variant Common-shape fields out of the API response and
	// writes any variant-specific Computed fields back into ``state`` (e.g. echoed URLs).
	Extract func(apiObj *P, state State) APIObject

	// CRUD method references — typically the bound ``(*api_client.APIClient).XyzConfig``
	// methods. The base calls these via the shared API client so per-variant files don't
	// hand-roll the error-wrapping plumbing.
	Create func(client *api_client.APIClient, payload P) (*P, error)
	Get    func(client *api_client.APIClient, templateName string) (*P, error)
	Update func(client *api_client.APIClient, templateName string, payload P) (*P, error)
	Delete func(client *api_client.APIClient, templateName string) error
}

func buildSchema[P any](spec Spec[P]) schema.Schema {
	attrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "Orca external service config identifier (UUID).",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"template_name": schema.StringAttribute{
			Required:      true,
			Description:   "Template name used as the URL key for update/delete. Changing this forces a new resource.",
			Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"is_enabled": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
		"is_default": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
	}
	if spec.SupportsBusinessUnits {
		attrs["business_units"] = schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Optional set of Orca business unit IDs that may use this integration.",
		}
	}
	for name, attribute := range spec.VariantAttributes {
		attrs[name] = attribute
	}
	return schema.Schema{Description: spec.Description, Attributes: attrs}
}

func applyCommon(ctx context.Context, st State, apiObj APIObject, supportsBUs bool, diags *diag.Diagnostics) {
	c := st.GetCommon()
	c.ID = types.StringValue(apiObj.ID)
	c.IsEnabled = types.BoolValue(apiObj.IsEnabled)
	c.IsDefault = types.BoolValue(apiObj.IsDefault)
	if apiObj.TemplateName != "" {
		c.TemplateName = types.StringValue(apiObj.TemplateName)
	}
	if supportsBUs {
		bus, busDiags := common.BusinessUnitsFromAPI(ctx, apiObj.BusinessUnits, c.BusinessUnits)
		diags.Append(busDiags...)
		c.BusinessUnits = bus
	} else {
		c.BusinessUnits = types.SetNull(types.StringType)
	}
	st.SetCommon(*c)
}

// genericResource implements resource.Resource for any Spec[P]. There is exactly one Create /
// Read / Update / Delete / Schema / Metadata / Configure / ImportState across all variants.
type genericResource[P any] struct {
	client *api_client.APIClient
	spec   Spec[P]
}

// New returns a fully configured resource.Resource. Per-variant files declare a Spec and
// hand it to this constructor; no extra resource type is required.
func New[P any](spec Spec[P]) resource.Resource {
	return &genericResource[P]{spec: spec}
}

func (r *genericResource[P]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.spec.TypeNameSuffix
}

func (r *genericResource[P]) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*api_client.APIClient)
}

func (r *genericResource[P]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = buildSchema(r.spec)
}

// errorWrap converts an api_client error into a TF diagnostic without forcing every variant
// to repeat the same AddError boilerplate.
func errorWrap(diags *diag.Diagnostics, action, ui string, err error) {
	diags.AddError(
		fmt.Sprintf("Error %s %s", action, ui),
		fmt.Sprintf("Could not %s %s: %s", action, ui, err.Error()),
	)
}

func (r *genericResource[P]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error creating %s", r.spec.UIName), "API client not configured.")
		return
	}
	plan := r.spec.NewState()
	resp.Diagnostics.Append(req.Plan.Get(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload := r.spec.BuildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.spec.Create(r.client, payload)
	if err != nil {
		errorWrap(&resp.Diagnostics, "creating", r.spec.UIName, err)
		return
	}
	applyCommon(ctx, plan, r.spec.Extract(created, plan), r.spec.SupportsBusinessUnits, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *genericResource[P]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := r.spec.NewState()
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	current, err := r.spec.Get(r.client, state.GetCommon().TemplateName.ValueString())
	if err != nil {
		errorWrap(&resp.Diagnostics, "reading", r.spec.UIName, err)
		return
	}
	if current == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	applyCommon(ctx, state, r.spec.Extract(current, state), r.spec.SupportsBusinessUnits, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *genericResource[P]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := r.spec.NewState()
	resp.Diagnostics.Append(req.Plan.Get(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state := r.spec.NewState()
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload := r.spec.BuildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.spec.Update(r.client, state.GetCommon().TemplateName.ValueString(), payload)
	if err != nil {
		errorWrap(&resp.Diagnostics, "updating", r.spec.UIName, err)
		return
	}
	applyCommon(ctx, plan, r.spec.Extract(updated, plan), r.spec.SupportsBusinessUnits, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *genericResource[P]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := r.spec.NewState()
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.spec.Delete(r.client, state.GetCommon().TemplateName.ValueString()); err != nil {
		errorWrap(&resp.Diagnostics, "deleting", r.spec.UIName, err)
	}
}

func (r *genericResource[P]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}

// Compile-time interface assertions.
var (
	_ resource.Resource                = &genericResource[struct{}]{}
	_ resource.ResourceWithConfigure   = &genericResource[struct{}]{}
	_ resource.ResourceWithImportState = &genericResource[struct{}]{}
)
