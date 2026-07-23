package shift_left_gitlab_group

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func policyIdsToTypes(refs []api_client.ScmPolicyRef) types.Set {
	if len(refs) == 0 {
		return types.SetNull(types.StringType)
	}
	elems := make([]attr.Value, 0, len(refs))
	for _, r := range refs {
		elems = append(elems, types.StringValue(r.ID))
	}
	return types.SetValueMust(types.StringType, elems)
}

// expandUpdate builds the PUT body from the model. policies = default_policies
// ? [] : ids. Adopt-existing writes go through shift_left_integration.Adopt,
// which also preserves live server state; this remains for the plain
// model->body path (and its unit test).
func expandUpdate(m *resourceModel) api_client.ScmInstallationUpdate {
	return shift_left_integration.ExpandUpdate(m.InstallationMode, m.DefaultPolicies, m.PoliciesIds, m.ConfigSettings)
}

func apiToState(inst *api_client.GitlabGroup) resourceModel {
	cs := shift_left_integration.FlattenConfigSettings(inst.ConfigSettings)
	return resourceModel{
		ID:               types.StringValue(inst.ID),
		InstallationID:   types.StringValue(inst.InstallationID),
		GroupID:          types.StringValue(inst.ID),
		AccountName:      types.StringValue(inst.AccountName),
		InstallationMode: types.StringValue(inst.InstallationMode),
		DefaultPolicies:  types.BoolValue(inst.DefaultPolicies),
		PoliciesIds:      policyIdsToTypes(inst.Policies),
		ProjectID:        types.StringValue(api_client.ProjectRefID(inst.Project)),
		ConfigSettings:   &cs,
	}
}
