// Package config_integration_common holds the resource skeleton shared by every
// /api/external_service/config integration (PagerDuty, Opsgenie, Snyk, Splunk, Cloudflare,
// Akamai, Zscaler ZPA, Terraform Cloud, Azure Sentinel). Each per-variant resource only has
// to supply the Variant interface (model + payload/response conversion + variant-specific
// schema attributes); CRUD plumbing, Metadata/Configure/ImportState, and the shared schema
// shell live here exactly once.
package config_integration_common

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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

// Common holds the cross-variant fields every config-endpoint integration carries.
type Common struct {
	ID            types.String
	TemplateName  types.String
	IsEnabled     types.Bool
	IsDefault     types.Bool
	BusinessUnits types.Set
}

// APIObject is the API-shape view of those cross-variant fields. Variants pull the variant-
// specific fields out of api_client structs themselves.
type APIObject struct {
	ID            string
	TemplateName  string
	IsEnabled     bool
	IsDefault     bool
	BusinessUnits []string
}

// Variant is what each per-resource package supplies. The methods convert between Terraform
// state and the variant-specific api_client payload structs without this package having to
// take a dependency on any of them.
type Variant interface {
	// TypeNameSuffix is appended to the provider type name (e.g. "_integration_pagerduty").
	TypeNameSuffix() string
	// UIName is used in error messages ("PagerDuty integration").
	UIName() string
	// Description appears at the top of the rendered docs.
	Description() string
	// SupportsBusinessUnits says whether this integration accepts a business_units field at
	// all. PagerDuty and Cloudflare don't; Opsgenie/Snyk/etc. do.
	SupportsBusinessUnits() bool
	// VariantAttributes returns the variant-specific TF schema attributes that get spliced
	// in alongside the shared (id/template_name/is_enabled/is_default/business_units) set.
	VariantAttributes() map[string]schema.Attribute
	// NewState constructs a zero-value plan/state model the framework can decode into.
	NewState() State
	// CRUD operations are wired through the variant so the base can call the right
	// api_client method without knowing which one.
	Create(client *api_client.APIClient, ctx context.Context, plan State, diags *diag.Diagnostics) (APIObject, diag.Diagnostics)
	Read(client *api_client.APIClient, ctx context.Context, state State, diags *diag.Diagnostics) (apiObj APIObject, found bool, errDiags diag.Diagnostics)
	Update(client *api_client.APIClient, ctx context.Context, plan State, templateName string, diags *diag.Diagnostics) (APIObject, diag.Diagnostics)
	Delete(client *api_client.APIClient, templateName string) error
}

// State is the per-variant TF model. Each variant package implements it to wire the framework
// codec into the variant-specific struct fields without exposing them to the base.
type State interface {
	GetCommon() *Common
	SetCommon(Common)
}

// Schema renders the full schema. “variant“ controls the variant-specific attributes; the
// rest (id/template_name/is_enabled/is_default and optional business_units) come from here.
func Schema(v Variant) schema.Schema {
	attrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "Orca external service config identifier (UUID).",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"template_name": schema.StringAttribute{
			Required:    true,
			Description: "Template name used as the URL key for update/delete. Changing this forces a new resource.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"is_enabled": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(true),
		},
		"is_default": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(false),
		},
	}
	if v.SupportsBusinessUnits() {
		attrs["business_units"] = schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Optional set of Orca business unit IDs that may use this integration.",
		}
	}
	for name, attribute := range v.VariantAttributes() {
		attrs[name] = attribute
	}
	return schema.Schema{
		Description: v.Description(),
		Attributes:  attrs,
	}
}

// ApplyCommon copies cross-variant fields from the API object back into state. “planned“
// is consulted so a null business_units stays null when the API echoes an empty list.
func ApplyCommon(ctx context.Context, st State, apiObj APIObject, supportsBUs bool, diags *diag.Diagnostics) {
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
		// Variants that don't expose business_units in their schema must keep this null so
		// the framework codec doesn't complain about an unknown attribute on save.
		c.BusinessUnits = types.SetNull(types.StringType)
	}
	st.SetCommon(*c)
}

// Resource is the embeddable CRUD base. Per-variant packages create one as:
//
//	type myResource struct { config_integration_common.Resource }
//	func New() resource.Resource { return &myResource{Resource: config_integration_common.Resource{V: &myVariant{}}} }
//	func (r *myResource) Schema(...) { resp.Schema = config_integration_common.Schema(r.V) }
type Resource struct {
	APIClient *api_client.APIClient
	V         Variant
}

// EnsureVariantAttrType keeps schema definitions of variant attributes terse — callers can
// use this to suppress the unused-import warning when only schema.* names are referenced.
// (Kept for parity with the per-variant files that previously imported attr/typehelpers).
var _ attr.Type = types.StringType

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.V.TypeNameSuffix()
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.APIClient = req.ProviderData.(*api_client.APIClient)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.APIClient == nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error creating %s", r.V.UIName()), "API client not configured.")
		return
	}

	plan := r.V.NewState()
	resp.Diagnostics.Append(req.Plan.Get(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := r.V.Create(r.APIClient, ctx, plan, &resp.Diagnostics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ApplyCommon(ctx, plan, apiObj, r.V.SupportsBusinessUnits(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := r.V.NewState()
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, found, diags := r.V.Read(r.APIClient, ctx, state, &resp.Diagnostics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	ApplyCommon(ctx, state, apiObj, r.V.SupportsBusinessUnits(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := r.V.NewState()
	resp.Diagnostics.Append(req.Plan.Get(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := r.V.NewState()
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := r.V.Update(r.APIClient, ctx, plan, state.GetCommon().TemplateName.ValueString(), &resp.Diagnostics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ApplyCommon(ctx, plan, apiObj, r.V.SupportsBusinessUnits(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := r.V.NewState()
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.V.Delete(r.APIClient, state.GetCommon().TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting %s", r.V.UIName()),
			fmt.Sprintf("Could not delete %s %s: %s", r.V.UIName(), state.GetCommon().TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
