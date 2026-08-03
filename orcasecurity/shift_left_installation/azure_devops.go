package shift_left_installation

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type azureInstallationModel struct {
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

func NewAzureDevopsInstallationResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_azure_devops_installation",
		SchemaFn:       azureInstallationSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
		},
		OpsFn: func(apiClient *api_client.APIClient) shift_left_integration.UnitOps {
			return shift_left_integration.InstallationLifecycle[azureInstallationModel, api_client.AzureDevopsInstallation]{
				SCMName: "Azure DevOps",
				Create: func(plan *azureInstallationModel) (*api_client.AzureDevopsInstallation, error) {
					return apiClient.CreateAzureDevopsInstallation(azureWriteBody(plan))
				},
				Get: apiClient.GetAzureDevopsInstallation,
				Update: func(plan *azureInstallationModel) (*api_client.AzureDevopsInstallation, error) {
					return apiClient.UpdateAzureDevopsInstallation(plan.ID.ValueString(), azureWriteBody(plan))
				},
				Delete:   apiClient.DeleteAzureDevopsInstallation,
				ID:       func(m *azureInstallationModel) string { return m.ID.ValueString() },
				SetState: azureSetState,
			}
		},
	}
}

func azureInstallationSchema() rschema.Schema {
	attrs := shift_left_integration.InstallationBaseAttrs("Azure DevOps", "https://dev.azure.com",
		"Azure DevOps personal access token.")
	attrs["account_name"] = rschema.StringAttribute{
		Optional: true,
		Description: "Azure DevOps organization name the token is scoped to. " +
			"Set it for a single-organization token; omit for an all-organizations token.",
	}
	attrs["access_token_type"] = shift_left_common.ComputedString(
		"Token scope as classified by Orca: `SINGLE_ACCOUNT` or `ALL_ACCOUNTS`.")
	attrs["access_token_account_name"] = shift_left_common.ComputedString(
		"Organization name the token is scoped to, as reported by the API.")
	return rschema.Schema{
		Description: "Connects an Azure DevOps server or organization to Orca Shift Left by registering a personal access token " +
			"(POST /api/shiftleft/azure_devops/installations/). The API never returns the token, so after `terraform import` " +
			"the next apply re-sends the configured token.",
		Attributes: attrs,
	}
}

func azureWriteBody(plan *azureInstallationModel) api_client.AzureDevopsInstallationWrite {
	return api_client.AzureDevopsInstallationWrite{
		Name:      plan.Name.ValueString(),
		ServerURL: plan.ServerURL.ValueString(),
		AccessTokenDetails: &api_client.AzureAccessTokenDetails{
			AccessToken: plan.AccessToken.ValueString(),
			AccountName: plan.AccountName.ValueString(),
		},
	}
}

func azureSetState(m *azureInstallationModel, api *api_client.AzureDevopsInstallation) {
	m.ID = types.StringValue(api.ID)
	m.Name = types.StringValue(api.Name)
	m.ServerURL = types.StringValue(api.ServerURL)
	m.AccessTokenType = types.StringValue(api.AccessTokenType)
	m.AccessTokenAccountName = types.StringValue(api.AccessTokenAccountName)
	m.ExternalServerURL = types.StringValue(api.ExternalServerURL)
	m.IntegrationStatus = types.StringValue(api.IntegrationStatus)
	m.CloudIntegration = types.BoolValue(api.CloudIntegration)
}

var azureInstallationsSpec = shift_left_common.ScmObjectListSpec[api_client.AzureDevopsInstallation]{
	TypeNameSuffix: "_shift_left_azure_devops_installations",
	Description: "Lists all Orca Azure DevOps shift-left server connections (installations) for fleet-wide for_each. " +
		"Use each installation's `id` as `installation_id` on `orcasecurity_shift_left_azure_devops_account` / repository resources.",
	CollectionKey:  "installations",
	ListErrorTitle: "Error listing Azure DevOps installations",
	Attrs: map[string]dschema.Attribute{
		"id":                        dschema.StringAttribute{Computed: true, Description: "Orca Azure DevOps installation UUID."},
		"name":                      dschema.StringAttribute{Computed: true},
		"server_url":                dschema.StringAttribute{Computed: true},
		"external_server_url":       dschema.StringAttribute{Computed: true},
		"access_token_type":         dschema.StringAttribute{Computed: true},
		"access_token_account_name": dschema.StringAttribute{Computed: true},
		"integration_status":        dschema.StringAttribute{Computed: true},
		"cloud_integration":         dschema.BoolAttribute{Computed: true},
	},
	AttrTypes: map[string]attr.Type{
		"id": types.StringType, "name": types.StringType, "server_url": types.StringType,
		"external_server_url": types.StringType, "access_token_type": types.StringType,
		"access_token_account_name": types.StringType, "integration_status": types.StringType,
		"cloud_integration": types.BoolType,
	},
	List: func(c *api_client.APIClient) ([]api_client.AzureDevopsInstallation, error) {
		return c.ListAzureDevopsInstallations()
	},
	Row: func(a *api_client.AzureDevopsInstallation) map[string]attr.Value {
		return map[string]attr.Value{
			"id":                        types.StringValue(a.ID),
			"name":                      types.StringValue(a.Name),
			"server_url":                tfconv.StringOrNull(a.ServerURL),
			"external_server_url":       tfconv.StringOrNull(a.ExternalServerURL),
			"access_token_type":         tfconv.StringOrNull(a.AccessTokenType),
			"access_token_account_name": tfconv.StringOrNull(a.AccessTokenAccountName),
			"integration_status":        tfconv.StringOrNull(a.IntegrationStatus),
			"cloud_integration":         types.BoolValue(a.CloudIntegration),
		}
	},
}

func NewAzureDevopsInstallationsDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(azureInstallationsSpec)
}
