package shift_left_github_installation

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func apiToState(inst *api_client.GithubInstallation) resourceModel {
	return resourceModel{
		ID:             types.StringValue(inst.ID),
		InstallationID: types.StringValue(inst.ID),
		ScmConfigFields: shift_left_integration.ScmConfigFieldsFromAPI(
			inst.AccountName, inst.IntegrationStatus, inst.InstallationMode,
			inst.DefaultPolicies, inst.Policies, inst.Project, inst.ConfigSettings,
		),
	}
}
