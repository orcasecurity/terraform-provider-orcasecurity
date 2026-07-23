package shift_left_bitbucket_installation

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &installationResource{}
	_ resource.ResourceWithConfigure   = &installationResource{}
	_ resource.ResourceWithImportState = &installationResource{}
)

type installationResource struct {
	apiClient *api_client.APIClient
}

func NewResource() resource.Resource { return &installationResource{} }

type resourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ServerURL         types.String `tfsdk:"server_url"`
	AccessToken       types.String `tfsdk:"access_token"`
	AccessTokenType   types.String `tfsdk:"access_token_type"`
	Username          types.String `tfsdk:"username"`
	AccountID         types.String `tfsdk:"account_id"`
	ExternalServerURL types.String `tfsdk:"external_server_url"`
	IntegrationStatus types.String `tfsdk:"integration_status"`
	CloudIntegration  types.Bool   `tfsdk:"cloud_integration"`
}

func (r *installationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_bitbucket_installation"
}

func (r *installationResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *installationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Connects a Bitbucket server or workspace to Orca Shift Left by registering an access token " +
			"(POST /api/shiftleft/bitbucket/installations/). The API never returns the token, so after `terraform import` " +
			"the next apply re-sends the configured token.",
		Attributes: map[string]rschema.Attribute{
			"id": rschema.StringAttribute{
				Computed:      true,
				Description:   "Installation UUID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": rschema.StringAttribute{
				Required:    true,
				Description: "Display name for the installation.",
			},
			"server_url": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Bitbucket server URL without a trailing slash. Omit for Bitbucket cloud (https://bitbucket.org).",
			},
			"access_token": rschema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Bitbucket access token. Write-only: never returned by the API.",
			},
			"access_token_type": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Token kind: `PAT` for a personal access token, `TOKEN` for a workspace (cloud) or project (server) token.",
				Validators: []validator.String{
					stringvalidator.OneOf("PAT", "TOKEN"),
				},
			},
			"username": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Bitbucket username owning the token (used with `PAT` tokens).",
			},
			"account_id": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Workspace or project slug the token is scoped to (used with `TOKEN` tokens).",
			},
			"external_server_url": rschema.StringAttribute{
				Computed:    true,
				Description: "Externally visible server URL, if different.",
			},
			"integration_status": rschema.StringAttribute{
				Computed:    true,
				Description: "Health status. Empty when healthy; `DISABLED_DUE_TO_INVALID_TOKEN` or `INSTALLATION_UNREACHABLE` otherwise.",
			},
			"cloud_integration": rschema.BoolAttribute{
				Computed:    true,
				Description: "True when connected to Bitbucket cloud.",
			},
		},
	}
}

func (r *installationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func writeBody(plan *resourceModel) api_client.BitbucketInstallationWrite {
	return api_client.BitbucketInstallationWrite{
		Name:      plan.Name.ValueString(),
		ServerURL: plan.ServerURL.ValueString(),
		AccessTokenDetails: &api_client.BitbucketAccessTokenDetails{
			AccessToken:     plan.AccessToken.ValueString(),
			AccessTokenType: plan.AccessTokenType.ValueString(),
			Username:        plan.Username.ValueString(),
			AccountID:       plan.AccountID.ValueString(),
		},
	}
}

func setState(m *resourceModel, api *api_client.BitbucketInstallation) {
	m.ID = types.StringValue(api.ID)
	m.Name = types.StringValue(api.Name)
	m.ServerURL = types.StringValue(api.ServerURL)
	m.ExternalServerURL = types.StringValue(api.ExternalServerURL)
	m.IntegrationStatus = types.StringValue(api.IntegrationStatus)
	m.CloudIntegration = types.BoolValue(api.CloudIntegration)
	if api.AccessTokenDetails != nil {
		m.AccessTokenType = types.StringValue(api.AccessTokenDetails.AccessTokenType)
		m.Username = types.StringValue(api.AccessTokenDetails.Username)
		m.AccountID = types.StringValue(api.AccessTokenDetails.AccountID)
	} else {
		m.AccessTokenType = types.StringValue("")
		m.Username = types.StringValue("")
		m.AccountID = types.StringValue("")
	}
}

func (r *installationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.apiClient.CreateBitbucketInstallation(writeBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Bitbucket installation", err.Error())
		return
	}
	setState(&plan, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *installationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	live, err := r.apiClient.GetBitbucketInstallation(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Bitbucket installation", err.Error())
		return
	}
	if live == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	setState(&state, live) // access_token stays as-is: the API never returns it
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *installationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.apiClient.UpdateBitbucketInstallation(plan.ID.ValueString(), writeBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Bitbucket installation", err.Error())
		return
	}
	setState(&plan, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *installationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apiClient.DeleteBitbucketInstallation(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting Bitbucket installation", err.Error())
	}
}
