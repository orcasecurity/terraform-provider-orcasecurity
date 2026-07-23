package shift_left_integration

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PolicyIDsFromSet extracts the non-null/non-unknown string element values from
// a policies_ids set.
func PolicyIDsFromSet(s types.Set) []string {
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

// PolicyIDsToSet builds a policies_ids set from a slice of ids, returning a null
// set for an empty/nil input.
func PolicyIDsToSet(ids []string) types.Set {
	if len(ids) == 0 {
		return types.SetNull(types.StringType)
	}
	elems := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		elems = append(elems, types.StringValue(id))
	}
	return types.SetValueMust(types.StringType, elems)
}

// ExpandUpdate builds the shared PUT body from resolved plan values.
// policies = default_policies ? [] : ids (matches preprocessAccountToUpdate in
// the UI).
func ExpandUpdate(mode types.String, defaultPolicies types.Bool, policiesIds types.Set, cfg *ConfigSettingsModel) api_client.ScmInstallationUpdate {
	ids := PolicyIDsFromSet(policiesIds)
	if defaultPolicies.ValueBool() {
		ids = []string{}
	}
	return api_client.ScmInstallationUpdate{
		InstallationMode: mode.ValueString(),
		DefaultPolicies:  defaultPolicies.ValueBool(),
		Policies:         ids,
		ConfigSettings:   ExpandConfigSettings(cfg),
	}
}

// ExistingUnit is the server-side snapshot of an integrated unit used to adopt
// it into Terraform without wiping server-managed state.
type ExistingUnit struct {
	InstallationMode string
	DefaultPolicies  bool
	PolicyIDs        []string
	ConfigSettings   api_client.ShiftLeftConfigSettings
	ProjectID        string
}

// ProjectIntent captures what the user's CONFIG (not the resolved plan) says
// about the project binding, so Adopt can distinguish "leave the binding
// unchanged" from "switch to policies". Plan values are unreliable here because
// UseStateForUnknown backfills omitted fields from prior state.
type ProjectIntent struct {
	// FromConfig is the project_id the user set in config; null/unknown when
	// the user omitted it. An empty-string value clears the binding.
	FromConfig types.String
	// PoliciesIntent is true when the user set policies_ids or default_policies
	// in config, i.e. explicitly chose policies over a project.
	PoliciesIntent bool
}

// ProjectIntentFrom builds a ProjectIntent from a resource's CONFIG values
// (config.project_id, config.policies_ids, config.default_policies).
func ProjectIntentFrom(configProjectID types.String, configPolicies types.Set, configDefault types.Bool) ProjectIntent {
	return ProjectIntent{
		FromConfig: configProjectID,
		PoliciesIntent: (!configPolicies.IsNull() && !configPolicies.IsUnknown()) ||
			(!configDefault.IsNull() && !configDefault.IsUnknown()),
	}
}

// Adopted holds the resolved plan values plus the PUT body for an
// adopt-existing write.
type Adopted struct {
	InstallationMode types.String
	DefaultPolicies  types.Bool
	PoliciesIds      types.Set
	ConfigSettings   *ConfigSettingsModel
	Body             api_client.ScmInstallationUpdate
}

// Adopt hydrates unset plan fields from the live unit so an adopt-existing apply
// always sends a complete body and never wipes server-managed state:
//   - configuration_settings: the user's overlay merged on top of live values
//     (the API requires every field present).
//   - installation_mode / default_policies: the live value when the user left
//     the field unset.
//   - policies: the live explicit policies when the user left policies_ids unset
//     (otherwise a bare `orcasecurity apply` would clear an adopted unit's
//     policies).
//   - project: resolved from ProjectIntent (bind / clear / preserve). Mirroring
//     the UI, the PUT sends project_id XOR policies, so policies is dropped when
//     the unit ends up project-bound.
func Adopt(planMode types.String, planDefault types.Bool, planPolicies types.Set, planConfig *ConfigSettingsModel, project ProjectIntent, ex ExistingUnit) Adopted {
	base := FlattenConfigSettings(ex.ConfigSettings)
	merged := MergeConfigSettings(base, planConfig)

	mode := planMode
	if mode.IsNull() || mode.IsUnknown() {
		mode = types.StringValue(ex.InstallationMode)
	}
	defaultPolicies := planDefault
	if defaultPolicies.IsNull() || defaultPolicies.IsUnknown() {
		defaultPolicies = types.BoolValue(ex.DefaultPolicies)
	}
	policies := planPolicies
	if policies.IsNull() || policies.IsUnknown() {
		policies = PolicyIDsToSet(ex.PolicyIDs)
	}

	// Resolve the effective project binding from config intent.
	projectID := ex.ProjectID
	switch {
	case project.PoliciesIntent:
		// User explicitly chose policies -> clear any project binding.
		projectID = ""
	case !project.FromConfig.IsNull() && !project.FromConfig.IsUnknown():
		// User set project_id explicitly ("" clears, non-empty binds).
		projectID = project.FromConfig.ValueString()
	default:
		// User touched neither -> preserve whatever the unit already has.
		projectID = ex.ProjectID
	}

	body := ExpandUpdate(mode, defaultPolicies, policies, &merged)
	if projectID != "" {
		body.ProjectID = projectID
		body.Policies = nil
	}

	return Adopted{
		InstallationMode: mode,
		DefaultPolicies:  defaultPolicies,
		PoliciesIds:      policies,
		ConfigSettings:   &merged,
		Body:             body,
	}
}
