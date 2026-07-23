package shift_left_gitlab_group

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func policyIdsFromTypes(vals []types.String) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if !v.IsNull() && !v.IsUnknown() {
			out = append(out, v.ValueString())
		}
	}
	return out
}

func policyIdsToTypes(refs []api_client.ScmPolicyRef) []types.String {
	if len(refs) == 0 {
		return nil
	}
	out := make([]types.String, 0, len(refs))
	for _, r := range refs {
		out = append(out, types.StringValue(r.ID))
	}
	return out
}

// expandUpdate builds the PUT body. policies = default_policies ? [] : ids.
func expandUpdate(m *resourceModel) api_client.ScmInstallationUpdate {
	ids := policyIdsFromTypes(m.PoliciesIds)
	if m.DefaultPolicies.ValueBool() {
		ids = []string{}
	}
	return api_client.ScmInstallationUpdate{
		InstallationMode: m.InstallationMode.ValueString(),
		DefaultPolicies:  m.DefaultPolicies.ValueBool(),
		Policies:         ids,
		ConfigSettings:   shift_left_integration.ExpandConfigSettings(m.ConfigSettings),
	}
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
		ConfigSettings:   &cs,
	}
}
