package shift_left_bitbucket_installation

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	attrs := shift_left_integration.InstallationBaseAttrs("Bitbucket", "https://bitbucket.org",
		"Bitbucket access token.")
	attrs["access_token_type"] = rschema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Token kind: `PAT` for a personal access token, `TOKEN` for a workspace (cloud) or project (server) token.",
		Validators: []validator.String{
			stringvalidator.OneOf("PAT", "TOKEN"),
		},
	}
	attrs["username"] = rschema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Bitbucket username owning the token (used with `PAT` tokens).",
	}
	attrs["account_id"] = rschema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Workspace or project slug the token is scoped to (used with `TOKEN` tokens).",
	}
	resp.Schema = rschema.Schema{
		Description: "Connects a Bitbucket server or workspace to Orca Shift Left by registering an access token " +
			"(POST /api/shiftleft/bitbucket/installations/). The API never returns the token, so after `terraform import` " +
			"the next apply re-sends the configured token.",
		Attributes: attrs,
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
	td := api.AccessTokenDetails
	if td == nil {
		td = &api_client.BitbucketAccessTokenDetails{}
	}
	m.AccessTokenType = types.StringValue(td.AccessTokenType)
	m.Username = types.StringValue(td.Username)
	m.AccountID = types.StringValue(td.AccountID)
}

func (r *installationResource) lifecycle() shift_left_integration.InstallationLifecycle[resourceModel, api_client.BitbucketInstallation] {
	return shift_left_integration.InstallationLifecycle[resourceModel, api_client.BitbucketInstallation]{
		SCMName: "Bitbucket",
		Create: func(plan *resourceModel) (*api_client.BitbucketInstallation, error) {
			return r.apiClient.CreateBitbucketInstallation(writeBody(plan))
		},
		Get: r.apiClient.GetBitbucketInstallation,
		Update: func(plan *resourceModel) (*api_client.BitbucketInstallation, error) {
			return r.apiClient.UpdateBitbucketInstallation(plan.ID.ValueString(), writeBody(plan))
		},
		Delete:   r.apiClient.DeleteBitbucketInstallation,
		ID:       func(m *resourceModel) string { return m.ID.ValueString() },
		SetState: setState,
	}
}

func (r *installationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.lifecycle().DoCreate(ctx, req, resp)
}

func (r *installationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.lifecycle().DoRead(ctx, req, resp)
}

func (r *installationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.lifecycle().DoUpdate(ctx, req, resp)
}

func (r *installationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.lifecycle().DoDelete(ctx, req, resp)
}
