package shift_left_github_installation

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var installationsSpec = shift_left_integration.ScmUnitListSpec[api_client.GithubInstallation]{
	TypeNameSuffix: "_shift_left_github_installations",
	Description:    "Lists all Orca GitHub shift-left installations for fleet-wide for_each.",
	CollectionKey:  "installations",
	ListErrorTitle: "Error listing GitHub installations",
	Extra: map[string]attr.Type{
		"id":                      types.StringType,
		"installation_id":         types.StringType,
		"github_installation_id":  types.Int64Type,
		"github_app_settings_url": types.StringType,
	},
	List: func(c *api_client.APIClient) ([]api_client.GithubInstallation, error) {
		return c.ListGithubInstallations()
	},
	Row: func(in *api_client.GithubInstallation) (string, api_client.ScmUnitCommonFields, map[string]attr.Value) {
		return in.AccountName, in.ScmUnitCommonFields, map[string]attr.Value{
			"id":                      types.StringValue(in.ID),
			"installation_id":         types.StringValue(in.ID),
			"github_installation_id":  types.Int64Value(in.GithubInstallationID),
			"github_app_settings_url": shift_left_integration.OptionalID(in.GithubAppSettingsURL),
		}
	},
}

func NewInstallationsDataSource() datasource.DataSource {
	return shift_left_integration.NewScmUnitListDataSource(installationsSpec)
}

func installationsToListValue(insts []api_client.GithubInstallation) (types.List, diag.Diagnostics) {
	return installationsSpec.ListValue(insts)
}
