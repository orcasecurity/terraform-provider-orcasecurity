package shift_left_github_account

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func apiToState(account *api_client.GithubInstallation) resourceModel {
	return resourceModel{
		ID:                   types.StringValue(account.ID),
		AccountID:            types.StringValue(account.ID),
		GithubInstallationID: types.Int64Value(account.GithubInstallationID),
		GithubAppSettingsURL: tfconv.StringOrNull(account.GithubAppSettingsURL),
		ScmConfigFields:      shift_left_integration.ScmConfigFieldsFromAPI(account.AccountName, account.ScmUnitCommonFields),
	}
}
