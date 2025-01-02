package business_unit

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	CloudProviders []types.String `tfsdk:"cloud_providers"`
	CustomTags     []types.String `tfsdk:"custom_tags"`
	CloudTags      []types.String `tfsdk:"cloud_tags"`
	AccountTags    []types.String `tfsdk:"cloud_account_tags"`
	CloudAccounts  []types.String `tfsdk:"cloud_account_ids"`
}

type businessUnitShiftLeftFilterModel struct {
	ShiftLeftProjects []types.String `tfsdk:"shiftleft_project_ids"`
}

type businessUnitResourceModel struct {
	ID              types.String                      `tfsdk:"id"`
	Name            types.String                      `tfsdk:"name"`
	Filter          *businessUnitFilterModel          `tfsdk:"filter_data"`
	ShiftLeftFilter *businessUnitShiftLeftFilterModel `tfsdk:"shiftleft_filter_data"`
	GlobalFilter    types.Bool                        `tfsdk:"global_filter"`
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
	/*if err != nil {
		resp.Diagnostics.AddError(
			"Error importing business unit",
			"Could not convert ID to int64: "+err.Error(),
		)
		return
	}*/

	businessUnit, err := r.apiClient.GetBusinessUnit(businessUnitId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing business unit",
			fmt.Sprintf("Could not get business unit with ID %s: %v", businessUnitId, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), businessUnitId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), businessUnit.Name)...)

	// Create a new state model
	state := businessUnitResourceModel{
		ID:   types.StringValue(businessUnitId),
		Name: types.StringValue(businessUnit.Name),
	}

	// Only set ShiftLeftFilter if it exists in the API response
	if businessUnit.ShiftLeftFilter != nil && len(businessUnit.ShiftLeftFilter.ShiftLeftProjects) > 0 {
		shiftLeftProjects := make([]types.String, len(businessUnit.ShiftLeftFilter.ShiftLeftProjects))
		for i, project := range businessUnit.ShiftLeftFilter.ShiftLeftProjects {
			shiftLeftProjects[i] = types.StringValue(project)
		}

		state.ShiftLeftFilter = &businessUnitShiftLeftFilterModel{
			ShiftLeftProjects: shiftLeftProjects,
		}
	}

	if businessUnit.Filter != nil {

		filter := &businessUnitFilterModel{}
		hasFilterData := false

		if len(businessUnit.Filter.CloudProviders) > 0 {
			filter.CloudProviders = make([]types.String, len(businessUnit.Filter.CloudProviders))
			for i, provider := range businessUnit.Filter.CloudProviders {
				filter.CloudProviders[i] = types.StringValue(provider)
			}
			hasFilterData = true
		}

		if len(businessUnit.Filter.CloudAccounts) > 0 {
			filter.CloudAccounts = make([]types.String, len(businessUnit.Filter.CloudAccounts))
			for i, account := range businessUnit.Filter.CloudAccounts {
				filter.CloudAccounts[i] = types.StringValue(account)
			}
			hasFilterData = true
		}

		if len(businessUnit.Filter.AccountTags) > 0 {
			filter.AccountTags = make([]types.String, len(businessUnit.Filter.AccountTags))
			for i, accountTags := range businessUnit.Filter.AccountTags {
				filter.AccountTags[i] = types.StringValue(accountTags)
			}
			hasFilterData = true
		}

		if len(businessUnit.Filter.CloudTags) > 0 {
			filter.CloudTags = make([]types.String, len(businessUnit.Filter.CloudTags))
			for i, cloudTags := range businessUnit.Filter.CloudTags {
				filter.CloudTags[i] = types.StringValue(cloudTags)
			}
			hasFilterData = true
		}

		if len(businessUnit.Filter.CustomTags) > 0 {
			filter.CustomTags = make([]types.String, len(businessUnit.Filter.CustomTags))
			for i, customTags := range businessUnit.Filter.CustomTags {
				filter.CustomTags[i] = types.StringValue(customTags)
			}
			hasFilterData = true
		}

		if hasFilterData {
			state.Filter = filter
		}
	}

	// Set the entire state at once
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *businessUnitResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a business unit. Please note that Shift Left business units are not yet supported in this Terraform provider. For more information, see the docs on [Business Units](https://docs.orcasecurity.io/docs/business-unit-feature).\n\nPlease note that a business unit cannot be composed of multiple, different filter types. You cannot compose 1 business unit that uses both cloud tags and custom tags, for example.",
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
				Description: "Whether or not this is a business unit all users within your Orca org can use. If set to true, then it is accessible to all other users in your org.",
				Optional:    true,
			},
			"shiftleft_filter_data": schema.SingleNestedAttribute{
				Description: "The filter to select Shift Left resources for the business unit. If you are creating a BU that only includes Shift Left resources (projects), this can be safely excluded.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"shiftleft_project_ids": schema.ListAttribute{
						Description: "A list of 1 or more Shift Left project IDs.",
						ElementType: types.StringType,
						Optional:    true,
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
					"cloud_account_ids": schema.ListAttribute{
						Description: "A list of 1 or more cloud account IDs.",
						ElementType: types.StringType,
						Optional:    true,
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

func generateCloudProviderFilter(plan *businessUnitFilterModel) (api_client.BusinessUnitFilter, diag.Diagnostics) {
	var filter api_client.BusinessUnitFilter
	var cpFilter = filter.CloudProviders
	var finalDiags diag.Diagnostics

	for _, item := range plan.CloudProviders {
		cpFilter = append(cpFilter, item.ValueString())
	}
	return api_client.BusinessUnitFilter{CloudProviders: cpFilter}, finalDiags
}

func generateCustomTagsFilter(plan *businessUnitFilterModel) (api_client.BusinessUnitFilter, diag.Diagnostics) {
	var filter api_client.BusinessUnitFilter
	var ctFilter = filter.CustomTags
	var finalDiags diag.Diagnostics

	for _, item := range plan.CustomTags {
		ctFilter = append(ctFilter, item.ValueString())
	}
	return api_client.BusinessUnitFilter{CustomTags: ctFilter}, finalDiags
}

func generateInventoryTagsFilter(plan *businessUnitFilterModel) (api_client.BusinessUnitFilter, diag.Diagnostics) {
	var filter api_client.BusinessUnitFilter
	var itFilter = filter.CloudTags
	var finalDiags diag.Diagnostics

	for _, item := range plan.CloudTags {
		itFilter = append(itFilter, item.ValueString())
	}
	return api_client.BusinessUnitFilter{CloudTags: itFilter}, finalDiags
}

func generateAccountTagsFilter(plan *businessUnitFilterModel) (api_client.BusinessUnitFilter, diag.Diagnostics) {
	var filter api_client.BusinessUnitFilter
	var atFilter = filter.AccountTags
	var finalDiags diag.Diagnostics

	for _, item := range plan.AccountTags {
		atFilter = append(atFilter, item.ValueString())
	}
	return api_client.BusinessUnitFilter{AccountTags: atFilter}, finalDiags
}

func generateCloudAccountsFilter(plan *businessUnitFilterModel) (api_client.BusinessUnitFilter, diag.Diagnostics) {
	var filter api_client.BusinessUnitFilter
	var aiFilter = filter.CloudAccounts
	var finalDiags diag.Diagnostics

	for _, item := range plan.CloudAccounts {
		aiFilter = append(aiFilter, item.ValueString())
	}
	return api_client.BusinessUnitFilter{CloudAccounts: aiFilter}, finalDiags
}

func generateShiftLeftProjectFilter(plan *businessUnitShiftLeftFilterModel) (api_client.BusinessUnitShiftLeftFilter, diag.Diagnostics) {
	var filter api_client.BusinessUnitShiftLeftFilter
	var slFilter = filter.ShiftLeftProjects
	var finalDiags diag.Diagnostics

	for _, item := range plan.ShiftLeftProjects {
		slFilter = append(slFilter, item.ValueString())
	}
	return api_client.BusinessUnitShiftLeftFilter{ShiftLeftProjects: slFilter}, finalDiags
}

func (r *businessUnitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan businessUnitResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Filter != nil && plan.Filter.CloudProviders != nil {
		filter, filterDiags := generateCloudProviderFilter(plan.Filter)
		diags.Append(filterDiags...)

		createReq := api_client.BusinessUnit{
			Name:   plan.Name.ValueString(),
			Filter: &filter,
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

	} else if plan.Filter != nil && plan.Filter.CloudAccounts != nil && (plan.ShiftLeftFilter == nil || plan.ShiftLeftFilter.ShiftLeftProjects == nil) {
		filter, filterDiags := generateCloudAccountsFilter(plan.Filter)
		diags.Append(filterDiags...)

		createReq := api_client.BusinessUnit{
			Name:   plan.Name.ValueString(),
			Filter: &filter,
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

	} else if plan.ShiftLeftFilter != nil && plan.ShiftLeftFilter.ShiftLeftProjects != nil && (plan.Filter == nil || plan.Filter.CloudAccounts == nil) {
		slFilter, _ := generateShiftLeftProjectFilter(plan.ShiftLeftFilter)
		createReq := api_client.BusinessUnit{
			Name:            plan.Name.ValueString(),
			ShiftLeftFilter: &slFilter,
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

	} else if plan.ShiftLeftFilter != nil && plan.ShiftLeftFilter.ShiftLeftProjects != nil && plan.Filter != nil && plan.Filter.CloudAccounts != nil {
		filter, filterDiags := generateCloudAccountsFilter(plan.Filter)
		diags.Append(filterDiags...)

		createReq := api_client.BusinessUnit{}

		if len(plan.ShiftLeftFilter.ShiftLeftProjects) > 0 {
			slFilter, _ := generateShiftLeftProjectFilter(plan.ShiftLeftFilter)
			createReq = api_client.BusinessUnit{
				Name:            plan.Name.ValueString(),
				ShiftLeftFilter: &slFilter,
				Filter:          &filter,
			}

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

	} else if plan.ShiftLeftFilter != nil && plan.ShiftLeftFilter.ShiftLeftProjects != nil && plan.Filter != nil && plan.Filter.CloudProviders != nil {
		filter, filterDiags := generateCloudProviderFilter(plan.Filter)
		diags.Append(filterDiags...)

		slFilter, _ := generateShiftLeftProjectFilter(plan.ShiftLeftFilter)
		createReq := api_client.BusinessUnit{
			Name:            plan.Name.ValueString(),
			ShiftLeftFilter: &slFilter,
			Filter:          &filter,
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

	} else if plan.Filter != nil && plan.Filter.CustomTags != nil {
		filter, filterDiags := generateCustomTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		createReq := api_client.BusinessUnit{
			Name:   plan.Name.ValueString(),
			Filter: &filter,
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

	} else if plan.Filter != nil && plan.Filter.AccountTags != nil {
		filter, filterDiags := generateAccountTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		createReq := api_client.BusinessUnit{
			Name:   plan.Name.ValueString(),
			Filter: &filter,
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

	} else if plan.Filter != nil && plan.Filter.CloudTags != nil {
		filter, filterDiags := generateInventoryTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		createReq := api_client.BusinessUnit{
			Name:   plan.Name.ValueString(),
			Filter: &filter,
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

	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *businessUnitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state businessUnitResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesBusinessUnitExist(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading business unit",
			fmt.Sprintf("Could not read business unit ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Business unit %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetBusinessUnit(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading business unit",
			fmt.Sprintf("Could not read business unit ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Error(ctx, instance.ID)
	state.Name = types.StringValue(instance.Name)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *businessUnitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan businessUnitResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
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

	if len(plan.Filter.CloudProviders) > 0 {
		filter, filterDiags := generateCloudProviderFilter(plan.Filter)
		diags.Append(filterDiags...)

		updateReq := api_client.BusinessUnit{
			Name:   plan.Name.ValueString(),
			Filter: &filter,
		}

		_, err := r.apiClient.UpdateBusinessUnit(plan.ID.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating business unit",
				"Could not update business unit, unexpected error: "+err.Error(),
			)
			return
		}

		_, err = r.apiClient.GetBusinessUnit(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading business unit",
				"Could not read Business Unit ID: "+plan.ID.ValueString()+": "+err.Error(),
			)
			return
		}

		diags = resp.State.Set(ctx, &plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if len(plan.ShiftLeftFilter.ShiftLeftProjects) > 0 && len(plan.Filter.CloudAccounts) == 0 {
		slFilter, _ := generateShiftLeftProjectFilter(plan.ShiftLeftFilter)
		updateReq := api_client.BusinessUnit{
			ID:              plan.ID.ValueString(),
			Name:            plan.Name.ValueString(),
			ShiftLeftFilter: &slFilter,
		}

		instance, err := r.apiClient.UpdateBusinessUnit(updateReq.ID, updateReq)

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating business unit",
				"Could not update business unit, unexpected error: "+err.Error(),
			)
			return
		}
		plan.ID = types.StringValue(instance.ID)

		diags = resp.State.Set(ctx, plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if len(plan.ShiftLeftFilter.ShiftLeftProjects) > 0 && len(plan.Filter.CloudAccounts) > 0 {
		filter, filterDiags := generateCloudAccountsFilter(plan.Filter)
		diags.Append(filterDiags...)
		slFilter, _ := generateShiftLeftProjectFilter(plan.ShiftLeftFilter)

		updateReq := api_client.BusinessUnit{
			Name:            plan.Name.ValueString(),
			ShiftLeftFilter: &slFilter,
			Filter:          &filter,
		}

		instance, err := r.apiClient.UpdateBusinessUnit(updateReq.ID, updateReq)

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating business unit",
				"Could not update business unit, unexpected error: "+err.Error(),
			)
			return
		}
		plan.ID = types.StringValue(instance.ID)

		diags = resp.State.Set(ctx, plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if len(plan.Filter.CustomTags) > 0 {
		filter, filterDiags := generateCustomTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		updateReq := api_client.BusinessUnit{
			Filter: &filter,
			Name:   plan.Name.ValueString(),
		}

		_, err := r.apiClient.UpdateBusinessUnit(plan.ID.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating business unit",
				"Could not update business unit, unexpected error: "+err.Error(),
			)
			return
		}

		_, err = r.apiClient.GetBusinessUnit(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading business unit",
				"Could not read Business Unit ID: "+plan.ID.ValueString()+": "+err.Error(),
			)
			return
		}

		diags = resp.State.Set(ctx, &plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if len(plan.Filter.CloudTags) > 0 {
		filter, filterDiags := generateInventoryTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		updateReq := api_client.BusinessUnit{
			Filter: &filter,
			Name:   plan.Name.ValueString(),
		}

		_, err := r.apiClient.UpdateBusinessUnit(plan.ID.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating business unit",
				"Could not update business unit, unexpected error: "+err.Error(),
			)
			return
		}

		_, err = r.apiClient.GetBusinessUnit(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading business unit",
				"Could not read Business Unit ID: "+plan.ID.ValueString()+": "+err.Error(),
			)
			return
		}

		diags = resp.State.Set(ctx, &plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if len(plan.Filter.AccountTags) > 0 {
		filter, filterDiags := generateAccountTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		updateReq := api_client.BusinessUnit{
			Filter: &filter,
			Name:   plan.Name.ValueString(),
		}

		_, err := r.apiClient.UpdateBusinessUnit(plan.ID.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating business unit",
				"Could not update business unit, unexpected error: "+err.Error(),
			)
			return
		}

		_, err = r.apiClient.GetBusinessUnit(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading business unit",
				"Could not read Business Unit ID: "+plan.ID.ValueString()+": "+err.Error(),
			)
			return
		}

		diags = resp.State.Set(ctx, &plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if len(plan.Filter.CloudAccounts) > 0 {
		filter, filterDiags := generateCloudAccountsFilter(plan.Filter)
		diags.Append(filterDiags...)

		updateReq := api_client.BusinessUnit{
			Filter: &filter,
			Name:   plan.Name.ValueString(),
		}

		_, err := r.apiClient.UpdateBusinessUnit(plan.ID.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating business unit",
				"Could not update business unit, unexpected error: "+err.Error(),
			)
			return
		}

		_, err = r.apiClient.GetBusinessUnit(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading business unit",
				"Could not read Business Unit ID: "+plan.ID.ValueString()+": "+err.Error(),
			)
			return
		}

		diags = resp.State.Set(ctx, &plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *businessUnitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state businessUnitResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
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
