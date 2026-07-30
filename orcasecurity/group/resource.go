package group

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

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
	_ resource.Resource                = &groupResource{}
	_ resource.ResourceWithConfigure   = &groupResource{}
	_ resource.ResourceWithImportState = &groupResource{}
)

type groupResource struct {
	apiClient *api_client.APIClient
}

type groupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	SSOGroup    types.Bool   `tfsdk:"sso_group"`
	Users       types.Set    `tfsdk:"users"`
}

func NewGroupResource() resource.Resource {
	return &groupResource{}
}

func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *groupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	//tflog.Error(ctx, "Setting up Schema")
	resp.Schema = schema.Schema{
		Description: "Provides a group resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Group ID.",
			},
			"name": schema.StringAttribute{
				Description: "Group name. Must be unique across your Orca org.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"sso_group": schema.BoolAttribute{
				Description: "SSO permissions group vs Orca-only. Create-time only; changing it replaces the resource.",
				Required:    true,
				PlanModifiers: []planmodifier.Bool{
					// Update API ignores sso_group.
					boolplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Group description.",
				Required:    true,
			},
			"users": schema.SetAttribute{
				Description: "Member user IDs (see the orcasecurity_users data source). API does not return membership on read; external removals are not detected.",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	users := userIDsFromSet(plan.Users)

	createReq := api_client.Group{
		Name:        plan.Name.ValueString(),
		SSOGroup:    plan.SSOGroup.ValueBool(),
		Description: plan.Description.ValueString(),
	}

	instance, err := r.apiClient.CreateGroup(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating group",
			"Could not create group, unexpected error: "+err.Error(),
		)
		return
	}
	plan.ID = types.StringValue(instance.ID)

	// Create API ignores users; set via AddGroupUsers.
	if err := r.apiClient.AddGroupUsers(instance.ID, users); err != nil {
		// The group already exists remotely even though membership failed to apply.
		// Persist the ID and known fields so Terraform keeps tracking it instead of
		// leaking it — the next apply would otherwise 400 on the name, which is
		// unique per org.
		plan.SSOGroup = types.BoolValue(instance.SSOGroup)
		plan.Users = types.SetNull(types.StringType)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Error adding group users",
			fmt.Sprintf("Group %s was created but its users could not be set: %s", instance.ID, err.Error()),
		)
		return
	}

	instance, err = r.apiClient.GetGroup(instance.ID)
	if err != nil {
		// Create and AddGroupUsers both already succeeded remotely; persist what we
		// know rather than leaking the group over a refresh failure.
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Error refreshing group",
			fmt.Sprintf("Group %s was created but could not be refreshed: %s", plan.ID.ValueString(), err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(setGroupStateFromAPI(ctx, &plan, instance, plan.Users)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func userIDsFromSet(s types.Set) []string {
	if s.IsNull() || s.IsUnknown() {
		return []string{}
	}
	var users []string
	for _, item := range s.Elements() {
		users = append(users, item.String()[1:len(item.String())-1])
	}
	return users
}

// GET group returns total_users only; keep plan/prior users when the API omits the member list.
func optionalUsersSetMatchPlan(ctx context.Context, planOrPrior types.Set, api []string) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(api) > 0 {
		return types.SetValueFrom(ctx, types.StringType, api)
	}
	if planOrPrior.IsUnknown() {
		return types.SetNull(types.StringType), diags
	}
	return planOrPrior, diags
}

func setGroupStateFromAPI(ctx context.Context, m *groupResourceModel, instance *api_client.Group, usersRef types.Set) diag.Diagnostics {
	var diags diag.Diagnostics
	m.SSOGroup = types.BoolValue(instance.SSOGroup)
	apiUsers := instance.Users
	if apiUsers == nil {
		apiUsers = []string{}
	}
	usersSet, d := optionalUsersSetMatchPlan(ctx, usersRef, apiUsers)
	diags.Append(d...)
	if !diags.HasError() {
		m.Users = usersSet
	}
	return diags
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesGroupExist(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading group",
			fmt.Sprintf("Could not read group ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Group %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetGroup(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading group",
			fmt.Sprintf("Could not read group ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.Description = types.StringValue(instance.Description)
	state.Name = types.StringValue(instance.Name)
	resp.Diagnostics.Append(setGroupStateFromAPI(ctx, &state, instance, state.Users)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var prior groupResourceModel
	diags = req.State.Get(ctx, &prior)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := api_client.Group{
		ID:          plan.ID.ValueString(),
		Name:        plan.Name.ValueString(),
		SSOGroup:    plan.SSOGroup.ValueBool(),
		Description: plan.Description.ValueString(),
	}

	if _, err := r.apiClient.UpdateGroup(updateReq); err != nil {
		resp.Diagnostics.AddError(
			"Error updating group",
			"Could not update group, unexpected error: "+err.Error(),
		)
		return
	}

	// The update endpoint's users field only replaces membership when non-empty and can
	// never clear it to zero (confirmed against the API), so the diff between prior and
	// planned members is applied explicitly through the add/remove subresource instead.
	priorUsers := userIDsFromSet(prior.Users)
	toAdd, toRemove := diffUserIDs(priorUsers, userIDsFromSet(plan.Users))

	if err := r.apiClient.AddGroupUsers(plan.ID.ValueString(), toAdd); err != nil {
		// Name/description already changed remotely; membership did not. Persist the
		// unchanged membership alongside the new name/description instead of losing them.
		plan.SSOGroup = types.BoolValue(updateReq.SSOGroup)
		plan.Users = prior.Users
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Error adding group users",
			fmt.Sprintf("Group %s was updated but new members could not be added: %s", plan.ID.ValueString(), err.Error()),
		)
		return
	}

	if err := r.apiClient.RemoveGroupUsers(plan.ID.ValueString(), toRemove); err != nil {
		// Additions above already landed; removals did not. Persist that partial
		// membership instead of the fully-planned set, which would overstate what
		// actually changed.
		appliedUsers, d := types.SetValueFrom(ctx, types.StringType, append(priorUsers, toAdd...))
		resp.Diagnostics.Append(d...)
		plan.SSOGroup = types.BoolValue(updateReq.SSOGroup)
		plan.Users = appliedUsers
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Error removing group users",
			fmt.Sprintf("Group %s was updated and new members added but old members could not be removed: %s", plan.ID.ValueString(), err.Error()),
		)
		return
	}

	instance, err := r.apiClient.GetGroup(plan.ID.ValueString())
	if err != nil {
		// Name/description and membership all already changed remotely; persist them
		// instead of losing track over a refresh failure.
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Error updating group",
			"Could not read group, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Description = types.StringValue(instance.Description)
	plan.Name = types.StringValue(instance.Name)
	resp.Diagnostics.Append(setGroupStateFromAPI(ctx, &plan, instance, plan.Users)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// diffUserIDs returns the members present in next but not prior (toAdd) and the members
// present in prior but not next (toRemove).
func diffUserIDs(prior, next []string) (toAdd, toRemove []string) {
	priorSet := make(map[string]bool, len(prior))
	for _, id := range prior {
		priorSet[id] = true
	}
	nextSet := make(map[string]bool, len(next))
	for _, id := range next {
		nextSet[id] = true
	}
	for _, id := range next {
		if !priorSet[id] {
			toAdd = append(toAdd, id)
		}
	}
	for _, id := range prior {
		if !nextSet[id] {
			toRemove = append(toRemove, id)
		}
	}
	return toAdd, toRemove
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteGroup(state.ID.String()[1 : len(state.ID.String())-1])
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting group",
			"Could not delete group, unexpected error: "+err.Error(),
		)
		return
	}
}
