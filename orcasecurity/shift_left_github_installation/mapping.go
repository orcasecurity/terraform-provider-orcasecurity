package shift_left_github_installation

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func apiToState(inst *api_client.GithubInstallation) resourceModel {
	return resourceModel{
		ID:                   types.StringValue(inst.ID),
		InstallationID:       types.StringValue(inst.ID),
		GithubInstallationID: types.Int64Value(inst.GithubInstallationID),
		GithubAppSettingsURL: shift_left_integration.OptionalID(inst.GithubAppSettingsURL),
		ScmConfigFields:      shift_left_integration.ScmConfigFieldsFromAPI(inst.AccountName, inst.ScmUnitCommonFields),
	}
}
