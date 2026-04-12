package group_access

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &groupAccessResource{}
	_ resource.ResourceWithConfigure   = &groupAccessResource{}
	_ resource.ResourceWithImportState = &groupAccessResource{}
)

type groupAccessResource struct {
	apiClient *api_client.APIClient
}

type groupAccessResourceModel struct {
	ID                types.String `tfsdk:"id"`
	GroupID           types.String `tfsdk:"group_id"`
	RoleID            types.String `tfsdk:"role_id"`
	AllCloudAccounts  types.Bool   `tfsdk:"all_cloud_accounts"`
	CloudAccounts     types.List   `tfsdk:"cloud_accounts"`
	ShiftleftProjects types.List   `tfsdk:"shiftleft_projects"`
	UserFilters       types.List   `tfsdk:"user_filters"`
}

func NewGroupAccessResource() resource.Resource {
	return &groupAccessResource{}
}

func (r *groupAccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_access"
}

func (r *groupAccessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *groupAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *groupAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns an RBAC role to a group with optional scope: all cloud accounts, specific cloud accounts, Shift Left projects, or user filters (business unit IDs from /api/filters). Backed by POST /api/rbac/access/group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Orca assignment id returned by the API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.StringAttribute{
				Description: "Target group id (from orcasecurity_group or the Orca UI).",
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

func stringSliceFromList(ctx context.Context, l types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if l.IsNull() || l.IsUnknown() {
		return []string{}, diags
	}
	var out []string
	diags = l.ElementsAs(ctx, &out, false)
	return out, diags
}

// optionalListMatchPlan maps API slices to Terraform optional lists: omitted config (null) stays null when API returns empty.
func optionalListMatchPlan(ctx context.Context, planOrPrior types.List, api []string) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(api) > 0 {
		return types.ListValueFrom(ctx, types.StringType, api)
	}
	if planOrPrior.IsNull() || planOrPrior.IsUnknown() {
		return types.ListNull(types.StringType), diags
	}
	return types.ListValueFrom(ctx, types.StringType, []string{})
}

func (r *groupAccessResource) modelToAPI(ctx context.Context, plan groupAccessResourceModel, assignmentID string) (api_client.GroupAccess, diag.Diagnostics) {
	var diags diag.Diagnostics
	cloudAccounts, d := stringSliceFromList(ctx, plan.CloudAccounts)
	diags.Append(d...)
	shiftleft, d := stringSliceFromList(ctx, plan.ShiftleftProjects)
	diags.Append(d...)
	userFilters, d := stringSliceFromList(ctx, plan.UserFilters)
	diags.Append(d...)
	if diags.HasError() {
		return api_client.GroupAccess{}, diags
	}
	return api_client.GroupAccess{
		ID:                assignmentID,
		AllCloudAccounts:  plan.AllCloudAccounts.ValueBool(),
		RoleID:            plan.RoleID.ValueString(),
		GroupID:           plan.GroupID.ValueString(),
		CloudAccounts:     cloudAccounts,
		ShiftleftProjects: shiftleft,
		UserFilters:       userFilters,
	}, diags
}

// apiToModel builds state from the API. ref is the plan (create/update) or prior state (read) for optional list null vs [] consistency.
func (r *groupAccessResource) apiToModel(ctx context.Context, ga *api_client.GroupAccess, ref *groupAccessResourceModel) (groupAccessResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	cloudList, d := optionalListMatchPlan(ctx, ref.CloudAccounts, ga.CloudAccounts)
	diags.Append(d...)
	shiftList, d := optionalListMatchPlan(ctx, ref.ShiftleftProjects, ga.ShiftleftProjects)
	diags.Append(d...)
	filterList, d := optionalListMatchPlan(ctx, ref.UserFilters, ga.UserFilters)
	diags.Append(d...)
	if diags.HasError() {
		return groupAccessResourceModel{}, diags
	}
	return groupAccessResourceModel{
		ID:                types.StringValue(ga.ID),
		GroupID:           types.StringValue(ga.GroupID),
		RoleID:            types.StringValue(ga.RoleID),
		AllCloudAccounts:  types.BoolValue(ga.AllCloudAccounts),
		CloudAccounts:     cloudList,
		ShiftleftProjects: shiftList,
		UserFilters:       filterList,
	}, diags
}

// mergeGroupAccessAfterCreate prefers GET-after-POST body; if missing, fills gaps from create response + plan + request payload.
func mergeGroupAccessAfterCreate(refreshed *api_client.GroupAccess, created *api_client.GroupAccess, plan groupAccessResourceModel, payload api_client.GroupAccess) *api_client.GroupAccess {
	if refreshed != nil {
		return refreshed
	}
	out := *created
	if out.GroupID == "" {
		out.GroupID = plan.GroupID.ValueString()
	}
	if out.RoleID == "" {
		out.RoleID = plan.RoleID.ValueString()
	}
	if out.CloudAccounts == nil {
		out.CloudAccounts = payload.CloudAccounts
	}
	if out.ShiftleftProjects == nil {
		out.ShiftleftProjects = payload.ShiftleftProjects
	}
	if out.UserFilters == nil {
		out.UserFilters = payload.UserFilters
	}
	out.AllCloudAccounts = payload.AllCloudAccounts
	return &out
}

func (r *groupAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diags := r.modelToAPI(ctx, plan, "")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreateGroupAccess(payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating group access", err.Error())
		return
	}

	matchWant, d := r.modelToAPI(ctx, plan, created.ID)
	resp.Diagnostics.Append(d...)
	var refreshed *api_client.GroupAccess
	if !resp.Diagnostics.HasError() {
		var err error
		refreshed, err = r.apiClient.FindGroupAccess(created.ID, matchWant)
		if err != nil {
			resp.Diagnostics.AddWarning("Could not read group access after create", err.Error())
		}
	}
	final := mergeGroupAccessAfterCreate(refreshed, created, plan, payload)

	state, diags := r.apiToModel(ctx, final, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *groupAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var prior groupAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	want, diags := r.modelToAPI(ctx, prior, prior.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ga, err := r.apiClient.FindGroupAccess(prior.ID.ValueString(), want)
	if err != nil {
		resp.Diagnostics.AddError("Error reading group access", fmt.Sprintf("Could not read id %s: %s", prior.ID.ValueString(), err.Error()))
		return
	}
	if ga == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	newState, diags := r.apiToModel(ctx, ga, &prior)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *groupAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var prior groupAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diags := r.modelToAPI(ctx, plan, prior.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdateGroupAccess(payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating group access", err.Error())
		return
	}

	state, diags := r.apiToModel(ctx, updated, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *groupAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteGroupAccess(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting group access", err.Error())
	}
}
