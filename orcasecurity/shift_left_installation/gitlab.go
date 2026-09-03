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

type gitlabInstallationModel struct {
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

func NewGitlabInstallationResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_gitlab_installation",
		SchemaFn:       gitlabInstallationSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
		},
		OpsFn: func(apiClient *api_client.APIClient) shift_left_integration.UnitOps {
			return shift_left_integration.InstallationLifecycle[gitlabInstallationModel, api_client.GitlabInstallation]{
				SCMName: "GitLab",
				Create: func(plan *gitlabInstallationModel) (*api_client.GitlabInstallation, error) {
					return apiClient.CreateGitlabInstallation(gitlabWriteBody(plan))
				},
				Get: apiClient.GetGitlabInstallation,
				Update: func(plan *gitlabInstallationModel) (*api_client.GitlabInstallation, error) {
					return apiClient.UpdateGitlabInstallation(plan.ID.ValueString(), gitlabWriteBody(plan))
				},
				Delete:   apiClient.DeleteGitlabInstallation,
				ID:       func(m *gitlabInstallationModel) string { return m.ID.ValueString() },
				SetState: gitlabSetState,
			}
		},
	}
}

func gitlabInstallationSchema() rschema.Schema {
	attrs := shift_left_integration.InstallationBaseAttrs("GitLab", "https://gitlab.com",
		"GitLab access token. Orca validates it on create, so it must be a valid group or personal access token.")
	attrs["read_only"] = shift_left_common.OptionalComputedBool(
		"Whether the token grants read-only access. Defaults to `false`. Must match the actual token permissions.")
	attrs["access_token_name"] = shift_left_common.ComputedString("Name of the token as reported by GitLab.")
	attrs["access_token_type"] = shift_left_common.ComputedString("Type of the token as reported by GitLab.")
	return rschema.Schema{
		Description: "Connects a GitLab server to Orca Shift Left by registering an access token " +
			"(POST /api/shiftleft/gitlab/installations/). Orca validates the token on create, so it must be a valid " +
			"group access token or personal access token. The API never returns the token, so after `terraform import` " +
			"the next apply re-sends the configured token.",
		Attributes: attrs,
	}
}

func gitlabWriteBody(plan *gitlabInstallationModel) api_client.GitlabInstallationWrite {
	return api_client.GitlabInstallationWrite{
		AccessToken: plan.AccessToken.ValueString(),
		Name:        plan.Name.ValueString(),
		ServerURL:   plan.ServerURL.ValueString(),
		// Always sent: the API resets an omitted read_only to false on PATCH.
		ReadOnly: plan.ReadOnly.ValueBool(),
	}
}

func gitlabSetState(m *gitlabInstallationModel, api *api_client.GitlabInstallation) {
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

var gitlabInstallationsSpec = shift_left_common.ScmObjectListSpec[api_client.GitlabInstallation]{
	TypeNameSuffix: "_shift_left_gitlab_installations",
	Description: "Lists all Orca GitLab shift-left server connections (installations) for fleet-wide for_each. " +
		"Use each installation's `id` as `installation_id` on `orcasecurity_shift_left_gitlab_group` / repository resources.",
	CollectionKey:  "installations",
	ListErrorTitle: "Error listing GitLab installations",
	Attrs: map[string]dschema.Attribute{
		"id":                  dschema.StringAttribute{Computed: true, Description: "Orca GitLab installation UUID."},
		"name":                dschema.StringAttribute{Computed: true},
		"server_url":          dschema.StringAttribute{Computed: true},
		"external_server_url": dschema.StringAttribute{Computed: true},
		"access_token_name":   dschema.StringAttribute{Computed: true},
		"access_token_type":   dschema.StringAttribute{Computed: true},
		"read_only":           dschema.BoolAttribute{Computed: true},
		"integration_status":  dschema.StringAttribute{Computed: true},
		"cloud_integration":   dschema.BoolAttribute{Computed: true},
	},
	AttrTypes: map[string]attr.Type{
		"id": types.StringType, "name": types.StringType, "server_url": types.StringType,
		"external_server_url": types.StringType, "access_token_name": types.StringType,
		"access_token_type": types.StringType, "read_only": types.BoolType,
		"integration_status": types.StringType, "cloud_integration": types.BoolType,
	},
	List: func(c *api_client.APIClient) ([]api_client.GitlabInstallation, error) {
		return c.ListGitlabInstallations()
	},
	Row: func(a *api_client.GitlabInstallation) map[string]attr.Value {
		return map[string]attr.Value{
			"id":                  types.StringValue(a.ID),
			"name":                types.StringValue(a.Name),
			"server_url":          tfconv.StringOrNull(a.ServerURL),
			"external_server_url": tfconv.StringOrNull(a.ExternalServerURL),
			"access_token_name":   tfconv.StringOrNull(a.AccessTokenName),
			"access_token_type":   tfconv.StringOrNull(a.AccessTokenType),
			"read_only":           types.BoolValue(a.ReadOnly),
			"integration_status":  tfconv.StringOrNull(a.IntegrationStatus),
			"cloud_integration":   types.BoolValue(a.CloudIntegration),
		}
	},
}

func NewGitlabInstallationsDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(gitlabInstallationsSpec)
}
