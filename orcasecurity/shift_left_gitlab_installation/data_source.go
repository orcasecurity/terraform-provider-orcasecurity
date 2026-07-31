package shift_left_gitlab_installation

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var installationsSpec = shift_left_integration.ScmObjectListSpec[api_client.GitlabInstallation]{
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

func NewInstallationsDataSource() datasource.DataSource {
	return shift_left_integration.NewScmObjectListDataSource(installationsSpec)
}

func installationsToListValue(rows []api_client.GitlabInstallation) (types.List, diag.Diagnostics) {
	return installationsSpec.ListValue(rows)
}
