package user_access

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &userAccessResource{}
	_ resource.ResourceWithConfigure   = &userAccessResource{}
	_ resource.ResourceWithImportState = &userAccessResource{}
)

type userAccessResource struct {
	apiClient *api_client.APIClient
}

type userAccessResourceModel struct {
	ID                types.String `tfsdk:"id"`
	UserID            types.String `tfsdk:"user_id"`
	RoleID            types.String `tfsdk:"role_id"`
	AllCloudAccounts  types.Bool   `tfsdk:"all_cloud_accounts"`
	CloudAccounts     types.List   `tfsdk:"cloud_accounts"`
	ShiftleftProjects types.List   `tfsdk:"shiftleft_projects"`
	UserFilters       types.List   `tfsdk:"user_filters"`
}

func NewUserAccessResource() resource.Resource {
	return &userAccessResource{}
}

func (r *userAccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_access"
}

func (r *userAccessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *userAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *userAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a single user's permissions by assigning an RBAC role to a user with optional scope: " +
			"all cloud accounts, specific cloud accounts, Shift Left projects, or user filters (business unit IDs from /api/filters). " +
			"Backed by /api/rbac/access/user. This is the per-user counterpart of `orcasecurity_group_access`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Orca assignment id returned by the API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "Target user id (from orcasecurity_add_users or the Orca UI).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				Description: "Role id (built-in or orcasecurity_custom_role id).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"all_cloud_accounts": schema.BoolAttribute{
				Description: "When true, scope includes all cloud accounts (see Orca RBAC semantics).",
				Required:    true,
			},
			"cloud_accounts": schema.ListAttribute{
				Description: "Scoped cloud account ids when not using all_cloud_accounts.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"shiftleft_projects": schema.ListAttribute{
				Description: "Scoped Shift Left project ids.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"user_filters": schema.ListAttribute{
				Description: "User filter ids (business units use filter ids from orcasecurity_business_unit / /api/filters).",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

func (r *userAccessResource) modelToAPI(ctx context.Context, plan userAccessResourceModel, assignmentID string) (api_client.UserAccess, diag.Diagnostics) {
	var diags diag.Diagnostics
	cloudAccounts, d := common.StringSliceFromList(ctx, plan.CloudAccounts)
	diags.Append(d...)
	shiftleft, d := common.StringSliceFromList(ctx, plan.ShiftleftProjects)
	diags.Append(d...)
	userFilters, d := common.StringSliceFromList(ctx, plan.UserFilters)
	diags.Append(d...)
	if diags.HasError() {
		return api_client.UserAccess{}, diags
	}
	return api_client.UserAccess{
		ID:                assignmentID,
		AllCloudAccounts:  plan.AllCloudAccounts.ValueBool(),
		RoleID:            plan.RoleID.ValueString(),
		UserID:            plan.UserID.ValueString(),
		CloudAccounts:     cloudAccounts,
		ShiftleftProjects: shiftleft,
		UserFilters:       userFilters,
	}, diags
}

// apiToModel builds state from the API. ref is the plan (create/update) or prior state (read) for optional list null vs [] consistency.
func (r *userAccessResource) apiToModel(ctx context.Context, ua *api_client.UserAccess, ref *userAccessResourceModel) (userAccessResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	cloudList, d := common.OptionalListMatchPlan(ctx, ref.CloudAccounts, ua.CloudAccounts)
	diags.Append(d...)
	shiftList, d := common.OptionalListMatchPlan(ctx, ref.ShiftleftProjects, ua.ShiftleftProjects)
	diags.Append(d...)
	filterList, d := common.OptionalListMatchPlan(ctx, ref.UserFilters, ua.UserFilters)
	diags.Append(d...)
	if diags.HasError() {
		return userAccessResourceModel{}, diags
	}
	return userAccessResourceModel{
		ID:                types.StringValue(ua.ID),
		UserID:            types.StringValue(ua.UserID),
		RoleID:            types.StringValue(ua.RoleID),
		AllCloudAccounts:  types.BoolValue(ua.AllCloudAccounts),
		CloudAccounts:     cloudList,
		ShiftleftProjects: shiftList,
		UserFilters:       filterList,
	}, diags
}

func (r *userAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diags := r.modelToAPI(ctx, plan, "")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreateUserAccess(payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user access", err.Error())
		return
	}

	// Re-read the canonical row so nested scope fields reflect the server.
	refreshed, err := r.apiClient.FindUserAccess(created.ID, payload)
	if err != nil {
		resp.Diagnostics.AddWarning("Could not read user access after create", err.Error())
	}
	final := created
	if refreshed != nil {
		final = refreshed
	}

	state, diags := r.apiToModel(ctx, final, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *userAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var prior userAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	want, diags := r.modelToAPI(ctx, prior, prior.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ua, err := r.apiClient.FindUserAccess(prior.ID.ValueString(), want)
	if err != nil {
		resp.Diagnostics.AddError("Error reading user access", fmt.Sprintf("Could not read id %s: %s", prior.ID.ValueString(), err.Error()))
		return
	}
	if ua == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	newState, diags := r.apiToModel(ctx, ua, &prior)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *userAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var prior userAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diags := r.modelToAPI(ctx, plan, prior.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdateUserAccess(payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating user access", err.Error())
		return
	}

	state, diags := r.apiToModel(ctx, updated, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *userAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteUserAccess(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting user access", err.Error())
	}
}
