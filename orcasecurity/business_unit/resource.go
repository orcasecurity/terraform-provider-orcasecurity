package business_unit

import (
	"context"
	"encoding/json"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &businessUnitResource{}
	_ resource.ResourceWithConfigure   = &businessUnitResource{}
	_ resource.ResourceWithImportState = &businessUnitResource{}
	//_ resource.ResourceWithConfigValidators = &businessUnitResource{}
)

type businessUnitResource struct {
	apiClient *api_client.APIClient
}

type businessUnitFilterModel struct {
	CloudProviders  []types.String `tfsdk:"cloud_providers"`
	CustomTags      []types.String `tfsdk:"custom_tags"`
	CloudTags       []types.String `tfsdk:"cloud_tags"`
	AccountTags     []types.String `tfsdk:"cloud_account_tags"`
	CloudAccounts   []types.String `tfsdk:"cloud_vendor_id"`
	CloudAccountIds []types.String `tfsdk:"cloud_account_ids"` // Deprecated: use cloud_vendor_id instead
}

type businessUnitShiftLeftFilterModel struct {
	ShiftLeftProjects []types.String `tfsdk:"shiftleft_project_ids"`
}

type businessUnitResourceModel struct {
	ID                  types.String                      `tfsdk:"id"`
	Name                types.String                      `tfsdk:"name"`
	Filter              *businessUnitFilterModel          `tfsdk:"filter_data"`
	ShiftLeftFilter     *businessUnitShiftLeftFilterModel `tfsdk:"shiftleft_filter_data"`
	Config              jsontypes.Normalized              `tfsdk:"config"`
	GlobalFilter        types.Bool                        `tfsdk:"global_filter"`
	BusinessCriticality types.String                      `tfsdk:"business_criticality"`
	OwnerTeam           types.String                      `tfsdk:"owner_team"`
	Application         types.String                      `tfsdk:"application"`
	ContactEmails       []types.String                    `tfsdk:"contact_emails"`
	DeploymentStages    []types.String                    `tfsdk:"deployment_stages"`
}

// uuidValidator validates that a string is a valid UUID.
type uuidValidator struct{}

func (v uuidValidator) Description(_ context.Context) string {
	return "value must be a valid UUID"
}

func (v uuidValidator) MarkdownDescription(_ context.Context) string {
	return "value must be a valid UUID (e.g. `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`)"
}

func (v uuidValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := uuid.Parse(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid UUID",
			"shiftleft_project_id must be a valid UUID: "+err.Error(),
		)
	}
}

func NewBusinessUnitResource() resource.Resource {
	return &businessUnitResource{}
}

func (r *businessUnitResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_business_unit"
}

func (r *businessUnitResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *businessUnitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	businessUnitId := req.ID

	businessUnit, err := r.apiClient.GetBusinessUnit(businessUnitId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing business unit",
			fmt.Sprintf("Could not get business unit with ID %s: %v", businessUnitId, err),
		)
		return
	}
	if businessUnit == nil {
		resp.Diagnostics.AddError(
			"Error importing business unit",
			fmt.Sprintf("Business unit with ID %s does not exist", businessUnitId),
		)
		return
	}

	state := businessUnitResourceModel{
		ID:     types.StringValue(businessUnitId),
		Name:   types.StringValue(businessUnit.Name),
		Config: jsontypes.NewNormalizedNull(),
	}
	resp.Diagnostics.Append(setScopeInState(&state, businessUnit.Config, true)...)
	setMetadataInState(&state, businessUnit)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *businessUnitResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a business unit. For more information, see the docs on [Business Units](https://docs.orcasecurity.io/docs/business-unit-feature).\n\nValues set across `filter_data` and `shiftleft_filter_data` are combined with OR: the business unit covers resources that match any of them. For AND or ALL rules, use `config` instead.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Business Unit ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Business Unit name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"global_filter": schema.BoolAttribute{
				Description: "Org-wide when true. Omitted on create defaults to global.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"business_criticality": schema.StringAttribute{
				Description: "Business criticality. Valid values: `low`, `medium`, `high`, `critical`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("low", "medium", "high", "critical"),
				},
			},
			"owner_team": schema.StringAttribute{
				Description: "Owning team or department for the business unit.",
				Optional:    true,
			},
			"application": schema.StringAttribute{
				Description: "Application or product line the business unit represents.",
				Optional:    true,
			},
			"contact_emails": schema.ListAttribute{
				Description: "Contact emails associated with the business unit. Up to 2 values.",
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(2),
				},
			},
			"deployment_stages": schema.ListAttribute{
				Description: "Deployment stages associated with the business unit. Up to 2 values.",
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(2),
				},
			},
			"config": schema.StringAttribute{
				Description: "The business unit rule as JSON in the Orca business unit config format, for example " +
					"`jsonencode({ and = [{ some = { CloudProviders = [\"aws\"] } }, { all = { CustomTags = [\"env|prod\"] } }] })`. " +
					"Use it for rules that `filter_data` and `shiftleft_filter_data` cannot express. Conflicts with both.",
				CustomType: jsontypes.NormalizedType{},
				Optional:   true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("filter_data"),
						path.MatchRoot("shiftleft_filter_data"),
					),
				},
			},
			"shiftleft_filter_data": schema.SingleNestedAttribute{
				Description: "The filter to select Shift Left resources for the business unit. If you are creating a BU that only includes Shift Left resources (projects), this can be safely excluded.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"shiftleft_project_ids": schema.ListAttribute{
						Description: "A list of 1 or more Shift Left project IDs (must be valid UUIDs).",
						ElementType: types.StringType,
						Optional:    true,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(uuidValidator{}),
						},
					},
				},
			},
			"filter_data": schema.SingleNestedAttribute{
				Description: "The filter to select the resources of the business unit. If you are creating a BU that only includes Shift Left resources (projects), this can be safely excluded.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"cloud_providers": schema.ListAttribute{
						Description: "A list of 1 or more cloud providers. Valid values are `alicloud`, `aws`, `azure`, `gcp`, `oci`, and `shiftleft`.",
						ElementType: types.StringType,
						Optional:    true,
					},
					"cloud_vendor_id": schema.ListAttribute{
						Description: "A list of 1 or more cloud vendor IDs.",
						ElementType: types.StringType,
						Optional:    true,
					},
					"cloud_account_ids": schema.ListAttribute{
						Description:        "A list of 1 or more cloud vendor IDs. Use cloud_vendor_id instead.",
						DeprecationMessage: "Use cloud_vendor_id instead. This attribute will be removed in a future version.",
						ElementType:        types.StringType,
						Optional:           true,
					},
					"cloud_account_tags": schema.ListAttribute{
						Description: "A list of 1 or more cloud account tags. The key and value should be separated by a vertical line (|), rather than a colon(:).",
						ElementType: types.StringType,
						Optional:    true,
					},
					"cloud_tags": schema.ListAttribute{
						Description: "A list of 1 or more cloud tags (for AWS and Azure) or labels (for GCP). The key and value should be separated by a vertical line (|), rather than a colon(:).",
						ElementType: types.StringType,
						Optional:    true,
					},
					"custom_tags": schema.ListAttribute{
						Description: "A list of 1 or more custom tags. The key and value should be separated by a vertical line (|), rather than a colon(:).",
						ElementType: types.StringType,
						Optional:    true,
					},
				},
			},
		},
	}
}

func getCloudVendorIds(plan *businessUnitFilterModel) []types.String {
	if plan == nil {
		return nil
	}
	if len(plan.CloudAccounts) > 0 {
		return plan.CloudAccounts
	}
	return plan.CloudAccountIds
}

func stringSliceToTypes(s []string) []types.String {
	out := make([]types.String, len(s))
	for i, v := range s {
		out[i] = types.StringValue(v)
	}
	return out
}

func typesSliceToStrings(in []types.String) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = v.ValueString()
	}
	return out
}

func applyMetadataToRequest(req *api_client.BusinessUnit, plan *businessUnitResourceModel) {
	if !plan.GlobalFilter.IsNull() && !plan.GlobalFilter.IsUnknown() {
		v := plan.GlobalFilter.ValueBool()
		req.GlobalFilter = &v
	}
	req.BusinessCriticality = plan.BusinessCriticality.ValueString()
	req.OwnerTeam = plan.OwnerTeam.ValueString()
	req.Application = plan.Application.ValueString()
	req.ContactEmails = typesSliceToStrings(plan.ContactEmails)
	req.DeploymentStages = typesSliceToStrings(plan.DeploymentStages)
}

func setMetadataInState(state *businessUnitResourceModel, instance *api_client.BusinessUnit) {
	if instance.GlobalFilter != nil {
		state.GlobalFilter = types.BoolValue(*instance.GlobalFilter)
	} else {
		state.GlobalFilter = types.BoolNull()
	}
	if instance.BusinessCriticality != "" {
		state.BusinessCriticality = types.StringValue(instance.BusinessCriticality)
	} else {
		state.BusinessCriticality = types.StringNull()
	}
	if instance.OwnerTeam != "" {
		state.OwnerTeam = types.StringValue(instance.OwnerTeam)
	} else {
		state.OwnerTeam = types.StringNull()
	}
	if instance.Application != "" {
		state.Application = types.StringValue(instance.Application)
	} else {
		state.Application = types.StringNull()
	}
	if len(instance.ContactEmails) > 0 {
		state.ContactEmails = stringSliceToTypes(instance.ContactEmails)
	} else {
		state.ContactEmails = nil
	}
	if len(instance.DeploymentStages) > 0 {
		state.DeploymentStages = stringSliceToTypes(instance.DeploymentStages)
	} else {
		state.DeploymentStages = nil
	}
}

func businessUnitRequest(plan *businessUnitResourceModel) (api_client.BusinessUnit, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := api_client.BusinessUnit{
		Name:   plan.Name.ValueString(),
		BUType: api_client.BusinessUnitTypeCombinedFilter,
	}
	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		req.Config = json.RawMessage(plan.Config.ValueString())
	} else {
		config, err := configFromModel(plan.Filter, plan.ShiftLeftFilter)
		if err != nil {
			diags.AddError("Error building business unit config", err.Error())
			return req, diags
		}
		req.Config = config
	}
	if req.Config == nil {
		diags.AddError(
			"Business unit has no scope",
			"Set at least one value in filter_data or shiftleft_filter_data, or set config.",
		)
		return req, diags
	}
	applyMetadataToRequest(&req, plan)
	return req, diags
}

// setScopeInState writes the API config into whichever representation the
// state already uses. keepUnrepresentable (import) stores a config that the
// filter blocks cannot express in the config attribute instead of dropping it.
func setScopeInState(state *businessUnitResourceModel, config json.RawMessage, keepUnrepresentable bool) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(config) == 0 {
		config = json.RawMessage("{}")
	}
	if !state.Config.IsNull() {
		state.Config = jsontypes.NewNormalizedValue(string(config))
		state.Filter, state.ShiftLeftFilter = nil, nil
		return diags
	}

	filter, shiftLeft, ok := modelFromConfig(config)
	if !ok && keepUnrepresentable {
		state.Config = jsontypes.NewNormalizedValue(string(config))
		state.Filter, state.ShiftLeftFilter = nil, nil
		return diags
	}
	if !ok {
		diags.AddWarning(
			"Business unit rule cannot be shown as filter_data",
			fmt.Sprintf("Business unit %s has a rule that filter_data and shiftleft_filter_data cannot express "+
				"(for example and, all or a hierarchy path), most likely edited outside Terraform. "+
				"The next apply replaces it with the rule in your configuration. "+
				"Use the config attribute to manage the full rule.", state.ID.ValueString()),
		)
	}
	state.Filter, state.ShiftLeftFilter = filter, shiftLeft
	return diags
}

func setGlobalFilterFromResponse(plan *businessUnitResourceModel, instance *api_client.BusinessUnit) diag.Diagnostics {
	var diags diag.Diagnostics
	if instance.GlobalFilter != nil {
		plan.GlobalFilter = types.BoolValue(*instance.GlobalFilter)
	} else if plan.GlobalFilter.IsUnknown() {
		diags.AddError(
			"Error reading business unit",
			fmt.Sprintf("Business unit ID %s did not report global_filter", instance.ID),
		)
	}
	return diags
}

func (r *businessUnitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan businessUnitResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, diags := businessUnitRequest(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.CreateBusinessUnit(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating business unit",
			"Could not create business unit, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	resp.Diagnostics.Append(setGlobalFilterFromResponse(&plan, instance)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *businessUnitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state businessUnitResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Preserve whether previous state used deprecated cloud_account_ids (for backward compat)
	hadDeprecatedCloudAccountIds := state.Filter != nil && len(state.Filter.CloudAccountIds) > 0

	instance, err := r.apiClient.GetBusinessUnit(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading business unit",
			fmt.Sprintf("Could not read business unit ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Business unit %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(instance.Name)
	resp.Diagnostics.Append(setScopeInState(&state, instance.Config, false)...)
	// When user used deprecated cloud_account_ids: populate only CloudAccountIds and leave
	// CloudAccounts empty so state matches config (avoids planned update to remove cloud_vendor_id).
	if hadDeprecatedCloudAccountIds && state.Filter != nil && len(state.Filter.CloudAccounts) > 0 {
		state.Filter.CloudAccountIds = state.Filter.CloudAccounts
		state.Filter.CloudAccounts = nil
	}
	setMetadataInState(&state, instance)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *businessUnitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan businessUnitResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update business unit, unexpected error: "+plan.ID.ValueString(),
		)
		return
	}

	updateReq, diags := businessUnitRequest(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// bu_type=combined_filter also converts a legacy filter BU in place.
	instance, err := r.apiClient.UpdateBusinessUnit(plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating business unit",
			"Could not update business unit, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(setGlobalFilterFromResponse(&plan, instance)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *businessUnitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state businessUnitResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteBusinessUnit(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting business unit",
			"Could not delete business unit, unexpected error: "+err.Error(),
		)
		return
	}
}
