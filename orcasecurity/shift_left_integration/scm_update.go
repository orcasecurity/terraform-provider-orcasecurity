package shift_left_integration

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func PolicyIDsFromSet(s types.Set) []string {
	return tfconv.SetToStringSlice(s)
}

func PolicyIDsToSet(ids []string) types.Set {
	return tfconv.StringSliceToSet(ids)
}

// The API still returns SCAN_ALL on old units but rejects it on update, and has
// no default for an absent mode (empty → 400), so both map to the safe default.
func normalizeInstallationMode(mode string) string {
	if mode == "SCAN_ALL" || mode == "" {
		return "SELECTED_REPOSITORIES"
	}
	return mode
}

func ExpandUpdate(mode types.String, defaultPolicies types.Bool, policiesIds types.Set, cfg *ConfigSettingsModel) api_client.ScmInstallationUpdate {
	ids := PolicyIDsFromSet(policiesIds)
	if defaultPolicies.ValueBool() {
		ids = []string{}
	}
	return api_client.ScmInstallationUpdate{
		InstallationMode: normalizeInstallationMode(mode.ValueString()),
		DefaultPolicies:  defaultPolicies.ValueBool(),
		Policies:         ids,
		ConfigSettings:   ExpandConfigSettings(cfg),
	}
}

type ExistingUnit struct {
	InstallationMode string
	DefaultPolicies  bool
	PolicyIDs        []string
	ConfigSettings   api_client.ShiftLeftConfigSettings
	ProjectID        string
	RepoCount        int64
}

// Plan values are unreliable for project intent because UseStateForUnknown backfills omitted fields.
type ProjectIntent struct {
	FromConfig     types.String
	PoliciesIntent bool
}

// hasExplicitPolicies reports an explicit policies_ids list in config. Only an
// explicit list is mutually exclusive with project_id; default_policies is not.
func hasExplicitPolicies(policies types.Set) bool {
	return tfconv.Known(policies)
}

func ProjectIntentFrom(configProjectID types.String, configPolicies types.Set) ProjectIntent {
	return ProjectIntent{
		FromConfig:     configProjectID,
		PoliciesIntent: hasExplicitPolicies(configPolicies),
	}
}

// defaultConfigSettings are the settings a newly integrated unit gets when the
// config does not specify them. The four enums match the backend's own defaults,
// but DisableScanPullRequests deliberately does not: GitHub's backend default is
// true, while the UI always sends false when integrating a new org/group/account
// and the other three providers default to false. Sending false here is what
// makes PR scanning enabled by default on every provider, matching the product.
//
// These are also the fallbacks ExpandConfigSettings uses for the required enums,
// which the API rejects when empty.
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

// CreateUnitBody is Adopt seeded with defaults for a brand-new unit, rather than
// from a live one.
func CreateUnitBody(mode types.String, planDefault types.Bool, planPolicies types.Set, planConfig *ConfigSettingsModel, project ProjectIntent) api_client.ScmInstallationUpdate {
	seed := ExistingUnit{
		InstallationMode: mode.ValueString(),
		DefaultPolicies:  !project.PoliciesIntent,
		ConfigSettings:   defaultConfigSettings(),
	}
	return Adopt(mode, planDefault, planPolicies, planConfig, project, seed)
}

// Adopt hydrates unset plan fields from the live unit (seed on create).
// PUT sends project_id XOR explicit policies; default_policies may accompany project_id.
func Adopt(planMode types.String, planDefault types.Bool, planPolicies types.Set, planConfig *ConfigSettingsModel, project ProjectIntent, ex ExistingUnit) api_client.ScmInstallationUpdate {
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

	projectID := ex.ProjectID
	switch {
	case !project.FromConfig.IsNull() && !project.FromConfig.IsUnknown():
		projectID = project.FromConfig.ValueString() // explicit project_id wins (may be "" to clear)
	case project.PoliciesIntent:
		projectID = ""
	}

	body := ExpandUpdate(mode, defaultPolicies, policies, &merged)
	if projectID != "" {
		body.ProjectID = projectID
		body.Policies = nil
	}
	return body
}
