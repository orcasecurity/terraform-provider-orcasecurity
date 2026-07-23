package shift_left_azure_devops_installation

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
	resp.Schema = rschema.Schema{
		Description: "Connects an Azure DevOps server or organization to Orca Shift Left by registering a personal access token " +
			"(POST /api/shiftleft/azure_devops/installations/). The API never returns the token, so after `terraform import` " +
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
				Description: "Azure DevOps server URL without a trailing slash. Omit for Azure DevOps cloud (https://dev.azure.com).",
			},
			"access_token": rschema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Azure DevOps personal access token. Write-only: never returned by the API.",
			},
			"account_name": rschema.StringAttribute{
				Optional: true,
				Description: "Azure DevOps organization name the token is scoped to. " +
					"Set it for a single-organization token; omit for an all-organizations token.",
			},
			"access_token_type": rschema.StringAttribute{
				Computed:    true,
				Description: "Token scope as classified by Orca: `SINGLE_ACCOUNT` or `ALL_ACCOUNTS`.",
			},
			"access_token_account_name": rschema.StringAttribute{
				Computed:    true,
				Description: "Organization name the token is scoped to, as reported by the API.",
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
				Description: "True when connected to Azure DevOps cloud.",
			},
		},
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

func (r *installationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.apiClient.CreateAzureDevopsInstallation(writeBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Azure DevOps installation", err.Error())
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
	live, err := r.apiClient.GetAzureDevopsInstallation(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Azure DevOps installation", err.Error())
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
	updated, err := r.apiClient.UpdateAzureDevopsInstallation(plan.ID.ValueString(), writeBody(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Azure DevOps installation", err.Error())
		return
	}
	if updated == nil {
		resp.Diagnostics.AddError("Error updating Azure DevOps installation", "installation disappeared after update")
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
	if err := r.apiClient.DeleteAzureDevopsInstallation(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting Azure DevOps installation", err.Error())
	}
}
