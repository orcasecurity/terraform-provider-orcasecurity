package shift_left_azure_devops_installation

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	ServerURL              types.String `tfsdk:"server_url"`
	AccessToken            types.String `tfsdk:"access_token"`
	AccountName            types.String `tfsdk:"account_name"`
	AccessTokenType        types.String `tfsdk:"access_token_type"`
	AccessTokenAccountName types.String `tfsdk:"access_token_account_name"`
	ExternalServerURL      types.String `tfsdk:"external_server_url"`
	IntegrationStatus      types.String `tfsdk:"integration_status"`
	CloudIntegration       types.Bool   `tfsdk:"cloud_integration"`
}

func (r *installationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_azure_devops_installation"
}

func (r *installationResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *installationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := shift_left_integration.InstallationBaseAttrs("Azure DevOps", "https://dev.azure.com",
		"Azure DevOps personal access token.")
	attrs["account_name"] = rschema.StringAttribute{
		Optional: true,
		Description: "Azure DevOps organization name the token is scoped to. " +
			"Set it for a single-organization token; omit for an all-organizations token.",
	}
	attrs["access_token_type"] = rschema.StringAttribute{
		Computed:    true,
		Description: "Token scope as classified by Orca: `SINGLE_ACCOUNT` or `ALL_ACCOUNTS`.",
	}
	attrs["access_token_account_name"] = rschema.StringAttribute{
		Computed:    true,
		Description: "Organization name the token is scoped to, as reported by the API.",
	}
	resp.Schema = rschema.Schema{
		Description: "Connects an Azure DevOps server or organization to Orca Shift Left by registering a personal access token " +
			"(POST /api/shiftleft/azure_devops/installations/). The API never returns the token, so after `terraform import` " +
			"the next apply re-sends the configured token.",
		Attributes: attrs,
	}
}

func (r *installationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func writeBody(plan *resourceModel) api_client.AzureDevopsInstallationWrite {
	return api_client.AzureDevopsInstallationWrite{
		Name:      plan.Name.ValueString(),
		ServerURL: plan.ServerURL.ValueString(),
		AccessTokenDetails: &api_client.AzureAccessTokenDetails{
			AccessToken: plan.AccessToken.ValueString(),
			AccountName: plan.AccountName.ValueString(),
		},
	}
}

func setState(m *resourceModel, api *api_client.AzureDevopsInstallation) {
	m.ID = types.StringValue(api.ID)
	m.Name = types.StringValue(api.Name)
	m.ServerURL = types.StringValue(api.ServerURL)
	m.AccessTokenType = types.StringValue(api.AccessTokenType)
	m.AccessTokenAccountName = types.StringValue(api.AccessTokenAccountName)
	m.ExternalServerURL = types.StringValue(api.ExternalServerURL)
	m.IntegrationStatus = types.StringValue(api.IntegrationStatus)
	m.CloudIntegration = types.BoolValue(api.CloudIntegration)
}

func (r *installationResource) lifecycle() shift_left_integration.InstallationLifecycle[resourceModel, api_client.AzureDevopsInstallation] {
	return shift_left_integration.InstallationLifecycle[resourceModel, api_client.AzureDevopsInstallation]{
		SCMName: "Azure DevOps",
		Create: func(plan *resourceModel) (*api_client.AzureDevopsInstallation, error) {
			return r.apiClient.CreateAzureDevopsInstallation(writeBody(plan))
		},
		Get: r.apiClient.GetAzureDevopsInstallation,
		Update: func(plan *resourceModel) (*api_client.AzureDevopsInstallation, error) {
			return r.apiClient.UpdateAzureDevopsInstallation(plan.ID.ValueString(), writeBody(plan))
		},
		Delete:   r.apiClient.DeleteAzureDevopsInstallation,
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
