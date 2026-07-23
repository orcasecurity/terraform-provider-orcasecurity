package shift_left_gitlab_installation

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	AccessToken       types.String `tfsdk:"access_token"`
	ServerURL         types.String `tfsdk:"server_url"`
	ReadOnly          types.Bool   `tfsdk:"read_only"`
	ExternalServerURL types.String `tfsdk:"external_server_url"`
	AccessTokenName   types.String `tfsdk:"access_token_name"`
	AccessTokenType   types.String `tfsdk:"access_token_type"`
	IntegrationStatus types.String `tfsdk:"integration_status"`
	CloudIntegration  types.Bool   `tfsdk:"cloud_integration"`
}

func (r *installationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_gitlab_installation"
}

func (r *installationResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *installationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Connects a GitLab server to Orca Shift Left by registering an access token " +
			"(POST /api/shiftleft/gitlab/installations/). Orca validates the token on create, so it must be a valid " +
			"group access token or personal access token. The API never returns the token, so after `terraform import` " +
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
			"access_token": rschema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "GitLab access token. Write-only: never returned by the API.",
			},
			"server_url": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "GitLab server URL without a trailing slash. Omit for GitLab cloud (https://gitlab.com).",
			},
			"read_only": rschema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the token grants read-only access. Defaults to `false`. Must match the actual token permissions.",
			},
			"external_server_url": rschema.StringAttribute{
				Computed:    true,
				Description: "Externally visible server URL, if different.",
			},
			"access_token_name": rschema.StringAttribute{
				Computed:    true,
				Description: "Name of the token as reported by GitLab.",
			},
			"access_token_type": rschema.StringAttribute{
				Computed:    true,
				Description: "Type of the token as reported by GitLab.",
			},
			"integration_status": rschema.StringAttribute{
				Computed:    true,
				Description: "Health status. Empty when healthy; `DISABLED_DUE_TO_INVALID_TOKEN` or `INSTALLATION_UNREACHABLE` otherwise.",
			},
			"cloud_integration": rschema.BoolAttribute{
				Computed:    true,
				Description: "True when connected to GitLab cloud.",
			},
		},
	}
}

func (r *installationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func writeBody(plan *resourceModel) api_client.GitlabInstallationWrite {
	return api_client.GitlabInstallationWrite{
		AccessToken: plan.AccessToken.ValueString(),
		Name:        plan.Name.ValueString(),
		ServerURL:   plan.ServerURL.ValueString(),
		// Always sent: the API resets an omitted read_only to false on PATCH.
		ReadOnly: plan.ReadOnly.ValueBool(),
	}
}

func setState(m *resourceModel, api *api_client.GitlabInstallation) {
	m.ID = types.StringValue(api.ID)
	m.Name = types.StringValue(api.Name)
	m.ServerURL = types.StringValue(api.ServerURL)
	m.ReadOnly = types.BoolValue(api.ReadOnly)
	m.ExternalServerURL = types.StringValue(api.ExternalServerURL)
	m.AccessTokenName = types.StringValue(api.AccessTokenName)
	m.AccessTokenType = types.StringValue(api.AccessTokenType)
	m.IntegrationStatus = types.StringValue(api.IntegrationStatus)
	m.CloudIntegration = types.BoolValue(api.CloudIntegration)
}

func (r *installationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.apiClient.CreateGitlabInstallation(writeBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitLab installation", err.Error())
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
	live, err := r.apiClient.GetGitlabInstallation(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab installation", err.Error())
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
	updated, err := r.apiClient.UpdateGitlabInstallation(plan.ID.ValueString(), writeBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating GitLab installation", err.Error())
		return
	}
	if updated == nil {
		resp.Diagnostics.AddError("Error updating GitLab installation", "installation disappeared after update")
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
	if err := r.apiClient.DeleteGitlabInstallation(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting GitLab installation", err.Error())
	}
}
