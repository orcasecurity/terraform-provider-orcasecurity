package shift_left_gitlab_group

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func apiToState(inst *api_client.GitlabGroup) resourceModel {
	cs := shift_left_integration.FlattenConfigSettings(inst.ConfigSettings)
	return resourceModel{
		ID:               types.StringValue(inst.ID),
		InstallationID:   types.StringValue(inst.InstallationID),
		GroupID:          types.StringValue(inst.ID),
		AccountName:      types.StringValue(inst.AccountName),
		InstallationMode: types.StringValue(inst.InstallationMode),
		DefaultPolicies:  types.BoolValue(inst.DefaultPolicies),
		PoliciesIds:      shift_left_integration.PolicyIDsFromRefs(inst.Policies),
		ProjectID:        types.StringValue(api_client.ProjectRefID(inst.Project)),
		ConfigSettings:   &cs,
	}
}
