package webhook

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

var (
	_ resource.Resource                = &webhookResource{}
	_ resource.ResourceWithConfigure   = &webhookResource{}
	_ resource.ResourceWithImportState = &webhookResource{}
)

type webhookResource struct {
	apiClient *api_client.APIClient
}

type webhookConfigModel struct {
	WebhookURL    types.String `tfsdk:"webhook_url"`
	Type          types.String `tfsdk:"type"`
	APIKey        types.String `tfsdk:"api_key"`
	BodyFields    types.List   `tfsdk:"body_fields"`
	CustomHeaders types.Map    `tfsdk:"custom_headers"`
}

type webhookResourceModel struct {
	ID            types.String        `tfsdk:"id"`
	TemplateName  types.String        `tfsdk:"template_name"`
	IsEnabled     types.Bool          `tfsdk:"is_enabled"`
	IsDefault     types.Bool          `tfsdk:"is_default"`
	BusinessUnits types.Set           `tfsdk:"business_units"`
	Config        *webhookConfigModel `tfsdk:"config"`
}

func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_webhook"
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

// customHeaderObjectType describes the per-value object Orca expects under each header key.
// The list-of-objects shape lets a single header carry multiple values.
func customHeaderObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"custom": types.StringType,
	}}
}

func customHeaderListType() types.ListType {
	return types.ListType{ElemType: customHeaderObjectType()}
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a Webhook integration in Orca. Creates an external service config of `service_name = \"webhook\"` so automations can fire HTTP callbacks to a customer-controlled endpoint.",
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
				Description: "Template name for the webhook integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the webhook integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default webhook configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
			"business_units": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional set of Orca business unit IDs that may use this integration. Order does not matter. Leave unset to make the integration available to all business units the caller can access.",
			},
			"config": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Webhook configuration.",
				Attributes: map[string]schema.Attribute{
					"webhook_url": schema.StringAttribute{
						Required:    true,
						Description: "Destination URL Orca posts events to.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"type": schema.StringAttribute{
						Required:    true,
						Description: "Webhook variant. One of: `common`, `torq`, `tines`, `opus`, `coralogix`, `panther`.",
						Validators: []validator.String{
							stringvalidator.OneOf("common", "torq", "tines", "opus", "coralogix", "panther"),
						},
					},
					"api_key": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Optional API key sent with each webhook request. Treated as sensitive.",
					},
					"body_fields": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Optional list of Orca alert fields to include in the webhook request body. Leave unset to send the default payload.",
					},
					"custom_headers": schema.MapAttribute{
						Optional:    true,
						ElementType: customHeaderListType(),
						Description: "Optional custom HTTP headers, keyed by header name. Each value is a list of `{ custom = \"<value>\" }` objects so a single header can carry multiple values.",
					},
				},
			},
		},
	}
}

func (r *webhookResource) buildPayload(ctx context.Context, plan webhookResourceModel, diags *diag.Diagnostics) api_client.WebhookExternalServiceConfig {
	payload := api_client.WebhookExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
	}

	if plan.Config != nil {
		payload.Config = api_client.WebhookResourceConfig{
			WebhookURL: plan.Config.WebhookURL.ValueString(),
			Type:       plan.Config.Type.ValueString(),
			APIKey:     plan.Config.APIKey.ValueString(),
		}

		if !plan.Config.BodyFields.IsNull() && !plan.Config.BodyFields.IsUnknown() {
			var fields []string
			diags.Append(plan.Config.BodyFields.ElementsAs(ctx, &fields, false)...)
			payload.Config.BodyFields = fields
		}

		if !plan.Config.CustomHeaders.IsNull() && !plan.Config.CustomHeaders.IsUnknown() {
			headers, headerDiags := customHeadersToAPI(ctx, plan.Config.CustomHeaders)
			diags.Append(headerDiags...)
			payload.Config.CustomHeaders = headers
		}
	}

	payload.BusinessUnits = common.BusinessUnitsToAPI(ctx, plan.BusinessUnits, diags)
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
	// The API omits ``body_fields`` from responses when empty, so ``fields`` arrives as nil.
	// Mirror whatever shape the user already had in state (null vs empty list) — otherwise a
	// state with ``[]`` would flip to null on every refresh and produce a permanent diff.
	if len(fields) == 0 {
		if planned.IsNull() {
			return types.ListNull(types.StringType), nil
		}
		emptyList, diags := types.ListValue(types.StringType, []attr.Value{})
		return emptyList, diags
	}
	return types.ListValueFrom(ctx, types.StringType, fields)
}

// applyAPITopLevelToPlan updates the non-config fields from the API response and leaves the
// nested “config“ block exactly as the user planned it. The Plugin Framework treats any
// parent of a sensitive attribute (here, “config“ wraps “api_key“) as sensitive, so any
// post-apply mismatch anywhere inside “config“ — even on a non-sensitive child the API
// normalises (URL trailing slash, header ordering, etc.) — triggers
// "inconsistent values for sensitive attribute". Trusting the plan side-steps the whole class
// of issues on Create/Update.
func (r *webhookResource) applyAPITopLevelToPlan(ctx context.Context, plan *webhookResourceModel, apiObj *api_client.WebhookExternalServiceConfig, diags *diag.Diagnostics) {
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

// applyAPIResponseToState refreshes the whole resource — including the “config“ block — from
// an API response. Used by Read, where we want to detect drift the next time the user runs
// “terraform plan“. “api_key“ is intentionally not overwritten: the API may strip or
// re-encode it, and we already store the user-supplied value in state.
func (r *webhookResource) applyAPIResponseToState(ctx context.Context, state *webhookResourceModel, apiObj *api_client.WebhookExternalServiceConfig, diags *diag.Diagnostics) {
	r.applyAPITopLevelToPlan(ctx, state, apiObj, diags)

	if state.Config == nil {
		state.Config = &webhookConfigModel{}
	}
	state.Config.WebhookURL = types.StringValue(apiObj.Config.WebhookURL)
	state.Config.Type = types.StringValue(apiObj.Config.Type)
	if state.Config.APIKey.IsUnknown() {
		if apiObj.Config.APIKey != "" {
			state.Config.APIKey = types.StringValue(apiObj.Config.APIKey)
		} else {
			state.Config.APIKey = types.StringNull()
		}
	}

	bodyFields, bfDiags := bodyFieldsFromAPI(ctx, apiObj.Config.BodyFields, state.Config.BodyFields)
	diags.Append(bfDiags...)
	state.Config.BodyFields = bodyFields

	headers, headerDiags := customHeadersFromAPI(ctx, apiObj.Config.CustomHeaders, state.Config.CustomHeaders)
	diags.Append(headerDiags...)
	state.Config.CustomHeaders = headers
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating Webhook integration", "API client not configured.")
		return
	}

	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.apiClient.CreateWebhookConfig(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Webhook integration",
			fmt.Sprintf("Could not create Webhook integration: %s", err.Error()),
		)
		return
	}

	r.applyAPITopLevelToPlan(ctx, &plan, created, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetWebhookConfigByTemplate(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Webhook integration",
			fmt.Sprintf("Could not read Webhook integration %s: %s", state.TemplateName.ValueString(), err.Error()),
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

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.apiClient.UpdateWebhookConfig(state.TemplateName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Webhook integration",
			fmt.Sprintf("Could not update Webhook integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	r.applyAPITopLevelToPlan(ctx, &plan, updated, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteWebhookConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Webhook integration",
			fmt.Sprintf("Could not delete Webhook integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Config endpoints look up integrations by template_name; import keys on that value.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
