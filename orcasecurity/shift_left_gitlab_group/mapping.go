package shift_left_gitlab_group

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func apiToState(inst *api_client.GitlabGroup) resourceModel {
	return resourceModel{
		ID:              types.StringValue(inst.ID),
		InstallationID:  types.StringValue(inst.InstallationID),
		GroupID:         types.StringValue(inst.ID),
		GitlabGroupID:   types.Int64Value(inst.GitlabGroupID),
		ScmConfigFields: shift_left_integration.ScmConfigFieldsFromAPI(inst.AccountName, inst.ScmUnitCommonFields),
	}
}
