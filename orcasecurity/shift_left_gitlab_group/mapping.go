package shift_left_gitlab_group

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func policyIdsFromSet(s types.Set) []string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	elems := s.Elements()
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		if v, ok := e.(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			out = append(out, v.ValueString())
		}
	}
	return out
}

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

// expandUpdate builds the PUT body. policies = default_policies ? [] : ids.
func expandUpdate(m *resourceModel) api_client.ScmInstallationUpdate {
	ids := policyIdsFromSet(m.PoliciesIds)
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
