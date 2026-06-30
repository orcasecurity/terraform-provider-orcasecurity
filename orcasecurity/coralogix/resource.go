package coralogix

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

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

var (
	_ resource.Resource                = &coralogixResource{}
	_ resource.ResourceWithConfigure   = &coralogixResource{}
	_ resource.ResourceWithImportState = &coralogixResource{}
)

type coralogixResource struct {
	apiClient *api_client.APIClient
}

type coralogixResourceModel struct {
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

func NewCoralogixResource() resource.Resource {
	return &coralogixResource{}
}

func (r *coralogixResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_coralogix"
}

func (r *coralogixResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func customHeaderObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"custom": types.StringType,
	}}
}

func customHeaderListType() types.ListType {
	return types.ListType{ElemType: customHeaderObjectType()}
}

func (r *coralogixResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a Coralogix integration in Orca. Orca stores Coralogix as a webhook variant — under the hood this resource creates an external service config of `service_name = \"webhook\"` with `type = \"coralogix\"`.",
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
				Description: "Template name for the Coralogix integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"webhook_url": schema.StringAttribute{
				Required:    true,
				Description: "Coralogix ingest URL Orca posts events to (for example, `https://coralogix.us`).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"api_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Coralogix API key sent with each request. Treated as sensitive.",
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
				Optional:    true,
				Computed:    true,
				Description: "Whether the Coralogix integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default Coralogix configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
			"business_units": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional set of Orca business unit IDs that may use this integration. Leave unset to make the integration available to all business units the caller can access.",
			},
		},
	}
}

func businessUnitsFromAPI(ctx context.Context, apiBus []string, planned types.Set) (types.Set, diag.Diagnostics) {
	if len(apiBus) == 0 && planned.IsNull() {
		return types.SetNull(types.StringType), nil
	}
	return types.SetValueFrom(ctx, types.StringType, apiBus)
}

func (r *coralogixResource) buildPayload(ctx context.Context, plan coralogixResourceModel, diags *diag.Diagnostics) api_client.WebhookExternalServiceConfig {
	payload := api_client.WebhookExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.WebhookResourceConfig{
			WebhookURL: plan.WebhookURL.ValueString(),
			Type:       api_client.CoralogixWebhookType,
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

	if !plan.BusinessUnits.IsNull() && !plan.BusinessUnits.IsUnknown() {
		var bus []string
		diags.Append(plan.BusinessUnits.ElementsAs(ctx, &bus, false)...)
		payload.BusinessUnits = bus
	}

	return payload
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

func customHeadersFromAPI(ctx context.Context, headers map[string][]api_client.WebhookCustomHeaderValue, planned types.Map) (types.Map, diag.Diagnostics) {
	listType := customHeaderListType()
	if len(headers) == 0 && planned.IsNull() {
		return types.MapNull(listType), nil
	}

	objType := customHeaderObjectType()
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

// applyAPITopLevelToPlan refreshes the non-secret, non-config fields from the API response.
// The webhook ``config`` block contains the sensitive ``api_key`` — the Plugin Framework
// flags any post-apply diff inside such a parent as "inconsistent sensitive attribute", so on
// Create/Update we keep the planned config exactly as the user submitted it and only sync the
// outer state.
func (r *coralogixResource) applyAPITopLevelToPlan(ctx context.Context, plan *coralogixResourceModel, apiObj *api_client.WebhookExternalServiceConfig, diags *diag.Diagnostics) {
	plan.ID = types.StringValue(apiObj.ID)
	plan.IsEnabled = types.BoolValue(apiObj.IsEnabled)
	plan.IsDefault = types.BoolValue(apiObj.IsDefault)
	if apiObj.TemplateName != "" {
		plan.TemplateName = types.StringValue(apiObj.TemplateName)
	}

	bus, busDiags := businessUnitsFromAPI(ctx, apiObj.BusinessUnits, plan.BusinessUnits)
	diags.Append(busDiags...)
	plan.BusinessUnits = bus
}

// applyAPIResponseToState refreshes the whole resource — including config fields — from the
// API. Used by Read so subsequent ``terraform plan`` calls can detect drift. ``api_key`` is
// intentionally not overwritten: the API may re-encode or strip it and we already keep the
// user-supplied value in state.
func (r *coralogixResource) applyAPIResponseToState(ctx context.Context, state *coralogixResourceModel, apiObj *api_client.WebhookExternalServiceConfig, diags *diag.Diagnostics) {
	r.applyAPITopLevelToPlan(ctx, state, apiObj, diags)

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

	headers, headerDiags := customHeadersFromAPI(ctx, apiObj.Config.CustomHeaders, state.CustomHeaders)
	diags.Append(headerDiags...)
	state.CustomHeaders = headers
}

func (r *coralogixResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Coralogix integration", "API client not configured.")
		return
	}

	var plan coralogixResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.apiClient.CreateCoralogixConfig(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Coralogix integration",
			fmt.Sprintf("Could not create Coralogix integration: %s", err.Error()),
		)
		return
	}

	r.applyAPITopLevelToPlan(ctx, &plan, created, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *coralogixResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state coralogixResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetCoralogixConfig(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Coralogix integration",
			fmt.Sprintf("Could not read Coralogix integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	if current == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.applyAPIResponseToState(ctx, &state, current, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *coralogixResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan coralogixResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state coralogixResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.apiClient.UpdateCoralogixConfig(state.TemplateName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Coralogix integration",
			fmt.Sprintf("Could not update Coralogix integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	r.applyAPITopLevelToPlan(ctx, &plan, updated, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *coralogixResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state coralogixResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteCoralogixConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Coralogix integration",
			fmt.Sprintf("Could not delete Coralogix integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *coralogixResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
