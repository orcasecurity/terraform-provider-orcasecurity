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
	CloudProvider []types.String `tfsdk:"cloud_provider"`
	CustomTags    []types.String `tfsdk:"custom_tags"`
	InventoryTags []types.String `tfsdk:"inventory_tags"`
	AccountTags   []types.String `tfsdk:"accounts_tags_info_list"`
	CloudAccounts []types.String `tfsdk:"cloud_vendor_id"`
}

type businessUnitResourceModel struct {
	ID           types.String             `tfsdk:"id"`
	Name         types.String             `tfsdk:"name"`
	Filter       *businessUnitFilterModel `tfsdk:"filter_data"`
	GlobalFilter types.Bool               `tfsdk:"global_filter"`
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
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *businessUnitResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	//tflog.Error(ctx, "Setting up Schema")
	resp.Schema = schema.Schema{
		Description: "Provides a Business Unit resource. Please note that Shift Left business units are not yet supported in this Terraform Provider.",
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
			//Not sure if the body of this one is OK - originally came from description
			"global_filter": schema.BoolAttribute{
				Description: "Not sure",
				Optional:    true,
			},
			"filter_data": schema.SingleNestedAttribute{
				Description: "The filter to select the resources of the business unit.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"cloud_provider": schema.ListAttribute{
						Description: "A list of at least 1 cloud provider, each provided as a string.",
						ElementType: types.StringType,
						Optional:    true,
					},
					"cloud_vendor_id": schema.ListAttribute{
						Description: "A list of at least 1 cloud account #s, each provided as a string.",
						ElementType: types.StringType,
						Optional:    true,
					},
					"accounts_tags_info_list": schema.ListAttribute{
						Description: "A list of at least 1 account tag, each provided as a string. The key and value should be separated by a vertical line (|), rather than a colon(:).",
						ElementType: types.StringType,
						Optional:    true,
					},
					"inventory_tags": schema.ListAttribute{
						Description: "A list of at least 1 cloud tag, each provided as a string. The key and value should be separated by a vertical line (|), rather than a colon(:).",
						ElementType: types.StringType,
						Optional:    true,
					},
					"custom_tags": schema.ListAttribute{
						Description: "A list of at least 1 custom tag, each provided as a string. The key and value should be separated by a vertical line (|), rather than a colon(:).",
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
	var cpFilter = filter.CloudProvider
	var finalDiags diag.Diagnostics

	for _, item := range plan.CloudProvider {
		cpFilter = append(cpFilter, item.ValueString())
	}
	return api_client.BusinessUnitFilter{CloudProvider: cpFilter}, finalDiags
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
	var itFilter = filter.InventoryTags
	var finalDiags diag.Diagnostics

	for _, item := range plan.InventoryTags {
		itFilter = append(itFilter, item.ValueString())
	}
	return api_client.BusinessUnitFilter{InventoryTags: itFilter}, finalDiags
}

func generateAccountTagsFilter(plan *businessUnitFilterModel) (api_client.BusinessUnitFilter, diag.Diagnostics) {
	var filter api_client.BusinessUnitFilter
	var atFilter = filter.AccountTags
	var finalDiags diag.Diagnostics

	for _, item := range plan.AccountTags {
		atFilter = append(atFilter, item.ValueString())
	}
	return api_client.BusinessUnitFilter{CloudProvider: atFilter}, finalDiags
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

func (r *businessUnitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan businessUnitResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(plan.Filter.CloudProvider) > 0 {
		filter, filterDiags := generateCloudProviderFilter(plan.Filter)
		diags.Append(filterDiags...)

		createReq := api_client.BusinessUnit{
			Name:   plan.Name.ValueString(),
			Filter: filter,
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

		diags = resp.State.Set(ctx, plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if len(plan.Filter.CustomTags) > 0 {
		filter, filterDiags := generateCustomTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		createReq := api_client.BusinessUnit{
			Name:   plan.Name.ValueString(),
			Filter: filter,
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

		diags = resp.State.Set(ctx, plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if len(plan.Filter.AccountTags) > 0 {
		filter, filterDiags := generateAccountTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		createReq := api_client.BusinessUnit{
			Name:   plan.Name.ValueString(),
			Filter: filter,
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

		diags = resp.State.Set(ctx, plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if len(plan.Filter.InventoryTags) > 0 {
		filter, filterDiags := generateInventoryTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		createReq := api_client.BusinessUnit{
			Name:   plan.Name.ValueString(),
			Filter: filter,
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

		diags = resp.State.Set(ctx, plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else if len(plan.Filter.CloudAccounts) > 0 {
		filter, filterDiags := generateCloudAccountsFilter(plan.Filter)
		diags.Append(filterDiags...)

		createReq := api_client.BusinessUnit{
			Name:   plan.Name.ValueString(),
			Filter: filter,
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

		diags = resp.State.Set(ctx, plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
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
	tflog.Error(ctx, state.ID.ValueString())
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

	if len(plan.Filter.CloudProvider) > 0 {
		filterQuery, filterDiags := generateCloudProviderFilter(plan.Filter)
		diags.Append(filterDiags...)

		updateReq := api_client.BusinessUnit{
			Filter: filterQuery,
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
	} else if len(plan.Filter.CustomTags) > 0 {
		filterQuery, filterDiags := generateCustomTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		updateReq := api_client.BusinessUnit{
			Filter: filterQuery,
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
	} else if len(plan.Filter.InventoryTags) > 0 {
		filterQuery, filterDiags := generateInventoryTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		updateReq := api_client.BusinessUnit{
			Filter: filterQuery,
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
		filterQuery, filterDiags := generateAccountTagsFilter(plan.Filter)
		diags.Append(filterDiags...)

		updateReq := api_client.BusinessUnit{
			Filter: filterQuery,
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
		filterQuery, filterDiags := generateCloudAccountsFilter(plan.Filter)
		diags.Append(filterDiags...)

		updateReq := api_client.BusinessUnit{
			Filter: filterQuery,
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
