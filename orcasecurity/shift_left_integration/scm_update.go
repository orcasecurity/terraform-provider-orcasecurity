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

// normalizeInstallationMode maps the legacy stored mode SCAN_ALL to
// SELECTED_REPOSITORIES. The API can still return SCAN_ALL on old units but
// rejects it on update; the UI applies the same remap on every write
// (commonAccountMappers.ts), so echoing the live value back verbatim would
// 400 on every apply for a legacy unit.
func normalizeInstallationMode(mode string) string {
	if mode == "SCAN_ALL" {
		return "SELECTED_REPOSITORIES"
	}
	return mode
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

// policiesIntent is the single definition of "the config explicitly chose
// policies over a project binding": policies_ids or default_policies was set.
// Shared by ProjectIntentFrom and the project_id plan modifier so the plan
// and the apply can never disagree about the user's intent.
func policiesIntent(policies types.Set, defaultPolicies types.Bool) bool {
	return (!policies.IsNull() && !policies.IsUnknown()) ||
		(!defaultPolicies.IsNull() && !defaultPolicies.IsUnknown())
}

// ProjectIntentFrom builds a ProjectIntent from a resource's CONFIG values
// (config.project_id, config.policies_ids, config.default_policies).
func ProjectIntentFrom(configProjectID types.String, configPolicies types.Set, configDefault types.Bool) ProjectIntent {
	return ProjectIntent{
		FromConfig:     configProjectID,
		PoliciesIntent: policiesIntent(configPolicies, configDefault),
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

// defaultConfigSettings mirrors the API/UI defaults applied when creating a
// unit with an empty configuration_settings overlay.
func defaultConfigSettings() api_client.ShiftLeftConfigSettings {
	return api_client.ShiftLeftConfigSettings{
		DisableScanPullRequests: false,
		CommentsOnPullRequests:  "ALWAYS",
		PrSummaryComment:        "ALWAYS",
		SkipCheckRuns:           "ALWAYS",
		ConfigFileSupport:       "ENABLED",
		PrSummaryAppendix:       "",
	}
}

// CreateUnitBody builds the POST integrate body for a missing unit. Mode must
// already be resolved (typically SCAN_ALL_INCLUDE_FUTURE). Unset
// default_policies defaults to true when the config did not choose policies;
// configuration_settings merge onto API defaults.
func CreateUnitBody(mode types.String, planDefault types.Bool, planPolicies types.Set, planConfig *ConfigSettingsModel, project ProjectIntent) Adopted {
	base := FlattenConfigSettings(defaultConfigSettings())
	merged := MergeConfigSettings(base, planConfig)

	defaultPolicies := planDefault
	if defaultPolicies.IsNull() || defaultPolicies.IsUnknown() {
		if project.PoliciesIntent {
			defaultPolicies = types.BoolValue(false)
		} else {
			defaultPolicies = types.BoolValue(true)
		}
	}
	policies := planPolicies
	if policies.IsNull() || policies.IsUnknown() {
		policies = PolicyIDsToSet(nil)
	}

	projectID := ""
	switch {
	case project.PoliciesIntent:
		projectID = ""
	case !project.FromConfig.IsNull() && !project.FromConfig.IsUnknown():
		projectID = project.FromConfig.ValueString()
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
		mode = types.StringValue(normalizeInstallationMode(ex.InstallationMode))
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
