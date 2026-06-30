// Package webhook_variant_common holds the shared resource implementation behind the
// "webhook variant" integrations (Coralogix, Torq, ...). Orca stores them under
// service_name="webhook" with config.type pinned to the variant name, so the per-variant
// resource files would otherwise reimplement the same Read/Create/Update plumbing
// (~190 lines each) — they share everything except the type-name suffix and the constant
// config.type value.
package webhook_variant_common

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

// ResourceModel is the shared TF model used by every webhook variant.
type ResourceModel struct {
	ID            types.String `tfsdk:"id"`
	TemplateName  types.String `tfsdk:"template_name"`
	WebhookURL    types.String `tfsdk:"webhook_url"`
	APIKey        types.String `tfsdk:"api_key"`
	BodyFields    types.List   `tfsdk:"body_fields"`
	CustomHeaders types.Map    `tfsdk:"custom_headers"`
	IsEnabled     types.Bool   `tfsdk:"is_enabled"`
	IsDefault     types.Bool   `tfsdk:"is_default"`
	BusinessUnits types.Set    `tfsdk:"business_units"`
}

// Variant is what each per-variant package supplies — the only delta between Coralogix and
// Torq (other than naming).
type Variant struct {
	TypeNameSuffix    string // for example "_integration_coralogix"
	UIName            string // for error messages
	WebhookConfigType string // value to pin on config.type (e.g. "coralogix", "torq")
	CreateFn          func(*api_client.APIClient, api_client.WebhookExternalServiceConfig) (*api_client.WebhookExternalServiceConfig, error)
	GetFn             func(*api_client.APIClient, string) (*api_client.WebhookExternalServiceConfig, error)
	UpdateFn          func(*api_client.APIClient, string, api_client.WebhookExternalServiceConfig) (*api_client.WebhookExternalServiceConfig, error)
	DeleteFn          func(*api_client.APIClient, string) error
}

func CustomHeaderObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"custom": types.StringType,
	}}
}

func customHeaderListType() types.ListType {
	return types.ListType{ElemType: CustomHeaderObjectType()}
}

// Schema returns the shared schema. Per-variant resources override Description and
// per-attribute prose.
func Schema(description, urlDescription, apiKeyDescription string) schema.Schema {
	return schema.Schema{
		Description: description,
		Attributes: map[string]schema.Attribute{
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
			"webhook_url": schema.StringAttribute{
				Required:    true,
				Description: urlDescription,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"api_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: apiKeyDescription,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"body_fields": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional list of Orca alert fields to include in the request body.",
			},
			"custom_headers": schema.MapAttribute{
				Optional:    true,
				ElementType: customHeaderListType(),
				Description: "Optional custom HTTP headers, keyed by header name. Each value is a list of `{ custom = \"<value>\" }` objects so a single header can carry multiple values.",
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
			"business_units": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional set of Orca business unit IDs that may use this integration.",
			},
		},
	}
}

func customHeadersToAPI(ctx context.Context, headers types.Map) (map[string][]api_client.WebhookCustomHeaderValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	raw := map[string]types.List{}
	diags.Append(headers.ElementsAs(ctx, &raw, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := make(map[string][]api_client.WebhookCustomHeaderValue, len(raw))
	for key, list := range raw {
		if list.IsNull() || list.IsUnknown() {
			continue
		}
		var objs []struct {
			Custom types.String `tfsdk:"custom"`
		}
		diags.Append(list.ElementsAs(ctx, &objs, false)...)
		if diags.HasError() {
			return nil, diags
		}
		values := make([]api_client.WebhookCustomHeaderValue, 0, len(objs))
		for _, o := range objs {
			values = append(values, api_client.WebhookCustomHeaderValue{Custom: o.Custom.ValueString()})
		}
		out[key] = values
	}
	return out, diags
}

func customHeadersFromAPI(headers map[string][]api_client.WebhookCustomHeaderValue, planned types.Map) (types.Map, diag.Diagnostics) {
	listType := customHeaderListType()
	if len(headers) == 0 && planned.IsNull() {
		return types.MapNull(listType), nil
	}

	objType := CustomHeaderObjectType()
	elements := make(map[string]attr.Value, len(headers))
	var diags diag.Diagnostics
	for key, values := range headers {
		objs := make([]attr.Value, 0, len(values))
		for _, v := range values {
			obj, objDiags := types.ObjectValue(objType.AttrTypes, map[string]attr.Value{
				"custom": types.StringValue(v.Custom),
			})
			diags.Append(objDiags...)
			objs = append(objs, obj)
		}
		list, listDiags := types.ListValue(objType, objs)
		diags.Append(listDiags...)
		elements[key] = list
	}
	if diags.HasError() {
		return types.MapNull(listType), diags
	}
	result, mapDiags := types.MapValue(listType, elements)
	diags.Append(mapDiags...)
	return result, diags
}

func bodyFieldsFromAPI(ctx context.Context, fields []string, planned types.List) (types.List, diag.Diagnostics) {
	if len(fields) == 0 {
		if planned.IsNull() {
			return types.ListNull(types.StringType), nil
		}
		emptyList, diags := types.ListValue(types.StringType, []attr.Value{})
		return emptyList, diags
	}
	return types.ListValueFrom(ctx, types.StringType, fields)
}

// Resource is the embeddable base for webhook-variant resources.
type Resource struct {
	APIClient *api_client.APIClient
	Variant   Variant
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.Variant.TypeNameSuffix
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.APIClient = req.ProviderData.(*api_client.APIClient)
}

func (r *Resource) buildPayload(ctx context.Context, plan ResourceModel, diags *diag.Diagnostics) api_client.WebhookExternalServiceConfig {
	payload := api_client.WebhookExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.WebhookResourceConfig{
			WebhookURL: plan.WebhookURL.ValueString(),
			Type:       r.Variant.WebhookConfigType,
			APIKey:     plan.APIKey.ValueString(),
		},
	}

	if !plan.BodyFields.IsNull() && !plan.BodyFields.IsUnknown() {
		var fields []string
		diags.Append(plan.BodyFields.ElementsAs(ctx, &fields, false)...)
		payload.Config.BodyFields = fields
	}

	if !plan.CustomHeaders.IsNull() && !plan.CustomHeaders.IsUnknown() {
		headers, headerDiags := customHeadersToAPI(ctx, plan.CustomHeaders)
		diags.Append(headerDiags...)
		payload.Config.CustomHeaders = headers
	}

	payload.BusinessUnits = common.BusinessUnitsToAPI(ctx, plan.BusinessUnits, diags)
	return payload
}

// applyAPITopLevel refreshes the non-secret, non-config fields from the API response. The
// nested “config“ block contains the sensitive “api_key“ — the Plugin Framework flags any
// post-apply diff inside such a parent as "inconsistent sensitive attribute", so on
// Create/Update we keep the planned config exactly as the user submitted it.
func (r *Resource) applyAPITopLevel(ctx context.Context, plan *ResourceModel, apiObj *api_client.WebhookExternalServiceConfig, diags *diag.Diagnostics) {
	plan.ID = types.StringValue(apiObj.ID)
	plan.IsEnabled = types.BoolValue(apiObj.IsEnabled)
	plan.IsDefault = types.BoolValue(apiObj.IsDefault)
	if apiObj.TemplateName != "" {
		plan.TemplateName = types.StringValue(apiObj.TemplateName)
	}

	bus, busDiags := common.BusinessUnitsFromAPI(ctx, apiObj.BusinessUnits, plan.BusinessUnits)
	diags.Append(busDiags...)
	plan.BusinessUnits = bus
}

// applyAPIFull refreshes the whole resource (including the config block) — used by Read so
// subsequent “terraform plan“ calls can detect drift. “api_key“ is intentionally not
// overwritten: the API may strip or re-encode it and we already keep the user-supplied
// value in state.
func (r *Resource) applyAPIFull(ctx context.Context, state *ResourceModel, apiObj *api_client.WebhookExternalServiceConfig, diags *diag.Diagnostics) {
	r.applyAPITopLevel(ctx, state, apiObj, diags)

	state.WebhookURL = types.StringValue(apiObj.Config.WebhookURL)
	if state.APIKey.IsUnknown() {
		if apiObj.Config.APIKey != "" {
			state.APIKey = types.StringValue(apiObj.Config.APIKey)
		} else {
			state.APIKey = types.StringNull()
		}
	}

	bodyFields, bfDiags := bodyFieldsFromAPI(ctx, apiObj.Config.BodyFields, state.BodyFields)
	diags.Append(bfDiags...)
	state.BodyFields = bodyFields

	headers, headerDiags := customHeadersFromAPI(apiObj.Config.CustomHeaders, state.CustomHeaders)
	diags.Append(headerDiags...)
	state.CustomHeaders = headers
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.APIClient == nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error creating %s", r.Variant.UIName), "API client not configured.")
		return
	}

	var plan ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.Variant.CreateFn(r.APIClient, payload)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating %s", r.Variant.UIName),
			fmt.Sprintf("Could not create %s: %s", r.Variant.UIName, err.Error()),
		)
		return
	}

	r.applyAPITopLevel(ctx, &plan, created, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.Variant.GetFn(r.APIClient, state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading %s", r.Variant.UIName),
			fmt.Sprintf("Could not read %s %s: %s", r.Variant.UIName, state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
	if current == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.applyAPIFull(ctx, &state, current, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.Variant.UpdateFn(r.APIClient, state.TemplateName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating %s", r.Variant.UIName),
			fmt.Sprintf("Could not update %s %s: %s", r.Variant.UIName, state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	r.applyAPITopLevel(ctx, &plan, updated, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.Variant.DeleteFn(r.APIClient, state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting %s", r.Variant.UIName),
			fmt.Sprintf("Could not delete %s %s: %s", r.Variant.UIName, state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
