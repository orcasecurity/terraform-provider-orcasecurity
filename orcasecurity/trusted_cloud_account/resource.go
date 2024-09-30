package trusted_cloud_account

import (
	"context"
	"fmt"
	"strconv"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &trustedCloudAccountResource{}
	_ resource.ResourceWithConfigure   = &trustedCloudAccountResource{}
	_ resource.ResourceWithImportState = &trustedCloudAccountResource{}
)

type trustedCloudAccountResource struct {
	apiClient *api_client.APIClient
}

type trustedCloudAccountResourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"account_name"`
	Description    types.String `tfsdk:"description"`
	CloudProvider  types.String `tfsdk:"cloud_provider"`
	CloudAccountID types.String `tfsdk:"cloud_provider_id"`
}

func NewTrustedCloudAccountResource() resource.Resource {
	return &trustedCloudAccountResource{}
}

func (r *trustedCloudAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trusted_cloud_account"
}

func (r *trustedCloudAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *trustedCloudAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *trustedCloudAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	//tflog.Error(ctx, "Setting up Schema")
	resp.Schema = schema.Schema{
		Description: "Provides a trusted cloud account resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Description: "Trusted cloud account ID.",
			},
			"account_name": schema.StringAttribute{
				Description: "Human-friendly name for the trusted cloud account.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "Cloud account description.",
				Required:    true,
			},
			"cloud_provider": schema.StringAttribute{
				Description: "Cloud Provider. Potential options are aws, azure, etc.",
				Required:    true,
			},
			"cloud_provider_id": schema.StringAttribute{
				Description: "Account ID for the cloud account.",
				Required:    true,
			},
		},
	}
}

func (r *trustedCloudAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan trustedCloudAccountResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := api_client.TrustedCloudAccount{
		Name:           plan.Name.ValueString(),
		CloudProvider:  plan.CloudProvider.ValueString(),
		CloudAccountID: plan.CloudAccountID.ValueString(),
		Description:    plan.Description.ValueString(),
	}

	instance, err := r.apiClient.CreateTrustedCloudAccount(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating trusted cloud account",
			"Could not create trusted cloud account, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.Int64Value(int64(instance.ID))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *trustedCloudAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state trustedCloudAccountResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesTrustedCloudAccountExist(strconv.Itoa(int(state.ID.ValueInt64())))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading cloud account",
			fmt.Sprintf("Could not read cloud account ID %s: %s", strconv.Itoa(int(state.ID.ValueInt64())), err.Error()),
		)
		return
	}

	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Cloud account %s is missing on the remote side.", strconv.Itoa(int(state.ID.ValueInt64()))))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetTrustedCloudAccount(strconv.Itoa(int(state.ID.ValueInt64())))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading cloud account",
			fmt.Sprintf("Could not read cloud account ID %s: %s", strconv.Itoa(int(state.ID.ValueInt64())), err.Error()),
		)
		return
	}

	state.ID = types.Int64Value(int64(instance.ID))
	state.Description = types.StringValue(instance.Description)
	state.Name = types.StringValue(instance.Name)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *trustedCloudAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan trustedCloudAccountResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := api_client.TrustedCloudAccount{
		ID:             int(plan.ID.ValueInt64()),
		Name:           plan.Name.ValueString(),
		CloudProvider:  plan.CloudProvider.ValueString(),
		CloudAccountID: plan.CloudAccountID.ValueString(),
		Description:    plan.Description.ValueString(),
	}

	_, err := r.apiClient.UpdateTrustedCloudAccount(updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating trusted cloud account",
			"Could not update trusted cloud account, unexpected error: "+err.Error(),
		)
		return
	}

	instance, err := r.apiClient.GetTrustedCloudAccount(strconv.Itoa(int(plan.ID.ValueInt64())))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating cloud account",
			"Could not read cloud account, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.Int64Value(int64(instance.ID))
	plan.Description = types.StringValue(instance.Description)
	plan.Name = types.StringValue(instance.Name)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *trustedCloudAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state trustedCloudAccountResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteTrustedCloudAccount(state.ID.String()[1 : len(state.ID.String())-1])
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting cloud account",
			"Could not delete cloud account, unexpected error: "+err.Error(),
		)
		return
	}
}
