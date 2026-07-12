package add_users

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                     = &addUsersResource{}
	_ resource.ResourceWithConfigure        = &addUsersResource{}
	_ resource.ResourceWithImportState      = &addUsersResource{}
	_ resource.ResourceWithConfigValidators = &addUsersResource{}
)

type addUsersResource struct {
	apiClient *api_client.APIClient
}

type addUsersResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Email             types.String `tfsdk:"email"`
	RoleID            types.String `tfsdk:"role_id"`
	Groups            types.List   `tfsdk:"groups"`
	AllCloudAccounts  types.Bool   `tfsdk:"all_cloud_accounts"`
	CloudAccounts     types.List   `tfsdk:"cloud_accounts"`
	UserFilters       types.List   `tfsdk:"user_filters"`
	ShiftleftProjects types.List   `tfsdk:"shiftleft_projects"`
	MFARequired       types.Bool   `tfsdk:"mfa_required"`
	ShouldSendEmail   types.Bool   `tfsdk:"should_send_email"`
	InviteLink        types.String `tfsdk:"invite_link"`
	Expired           types.Bool   `tfsdk:"expired"`
}

func NewAddUsersResource() resource.Resource {
	return &addUsersResource{}
}

func (r *addUsersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_add_users"
}

func (r *addUsersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *addUsersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *addUsersResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("role_id"),
			path.MatchRoot("groups"),
		),
	}
}

func (r *addUsersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Invites a single user to the organization (\"Add Users\" in the UI). Backed by /api/user_invites. " +
			"Assign either an RBAC role (`role_id`) or one or more groups (`groups`). The Orca invite API has no update " +
			"operation, so changing any argument replaces the invite. Grant per-user permissions on an existing user with " +
			"`orcasecurity_user_access`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Invite id returned by the API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				Description: "Email address of the user to invite.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				Description: "RBAC role id to grant on registration. Mutually exclusive with `groups`.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"groups": schema.ListAttribute{
				Description: "RBAC group ids to add the user to on registration. Mutually exclusive with `role_id`.",
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"all_cloud_accounts": schema.BoolAttribute{
				Description: "When true, the granted role applies to all cloud accounts. Only relevant with `role_id`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"cloud_accounts": schema.ListAttribute{
				Description: "Cloud account ids the role applies to when not using `all_cloud_accounts`.",
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"user_filters": schema.ListAttribute{
				Description: "User filter ids (business units use filter ids from orcasecurity_business_unit / /api/filters).",
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"shiftleft_projects": schema.ListAttribute{
				Description: "Shift Left project ids the role applies to.",
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"mfa_required": schema.BoolAttribute{
				Description: "Require the invited user to set up MFA after registration.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"should_send_email": schema.BoolAttribute{
				Description: "Send an invitation email to the user. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"invite_link": schema.StringAttribute{
				Description: "Registration link for the invited user. Only populated at creation time.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expired": schema.BoolAttribute{
				Description: "Whether the invite has expired.",
				Computed:    true,
			},
		},
	}
}

func (r *addUsersResource) modelToRequest(ctx context.Context, plan addUsersResourceModel) (api_client.UserInviteRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	groups, d := common.StringSliceFromList(ctx, plan.Groups)
	diags.Append(d...)
	cloudAccounts, d := common.StringSliceFromList(ctx, plan.CloudAccounts)
	diags.Append(d...)
	userFilters, d := common.StringSliceFromList(ctx, plan.UserFilters)
	diags.Append(d...)
	shiftleft, d := common.StringSliceFromList(ctx, plan.ShiftleftProjects)
	diags.Append(d...)
	if diags.HasError() {
		return api_client.UserInviteRequest{}, diags
	}
	return api_client.UserInviteRequest{
		InviteUserEmails:  []string{plan.Email.ValueString()},
		RoleID:            plan.RoleID.ValueString(),
		Groups:            groups,
		AllCloudAccounts:  plan.AllCloudAccounts.ValueBool(),
		CloudAccounts:     cloudAccounts,
		UserFilters:       userFilters,
		ShiftleftProjects: shiftleft,
		MFARequired:       plan.MFARequired.ValueBool(),
		ShouldSendEmail:   plan.ShouldSendEmail.ValueBool(),
	}, diags
}

func (r *addUsersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan addUsersResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diags := r.modelToRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	invite, err := r.apiClient.CreateUserInvite(payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user invite", err.Error())
		return
	}

	plan.ID = types.StringValue(invite.ID)
	plan.InviteLink = types.StringValue(invite.InviteLink)
	plan.Expired = types.BoolValue(invite.Expired)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *addUsersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state addUsersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	invite, err := r.apiClient.GetUserInvite(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading user invite", err.Error())
		return
	}
	// A missing invite means the user registered or the invite was revoked.
	if invite == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if invite.Email != "" {
		state.Email = types.StringValue(invite.Email)
	}
	state.Expired = types.BoolValue(invite.Expired)
	// invite_link is only returned at creation time; keep the stored value.

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update never performs real work: every configurable argument forces replacement
// because the invite API has no update operation. It only reconciles computed
// state to satisfy the resource interface.
func (r *addUsersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan addUsersResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *addUsersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state addUsersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteUserInvite(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting user invite", err.Error())
	}
}
