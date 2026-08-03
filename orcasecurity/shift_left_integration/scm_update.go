package shift_left_integration

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// normalizeInstallationMode is write-side only: the API still returns SCAN_ALL on
// old units but rejects it on update, and has no default for an absent mode
// (empty → 400), so both map to the safe default on the wire.
func normalizeInstallationMode(mode string) string {
	if mode == "SCAN_ALL" || mode == "" {
		return "SELECTED_REPOSITORIES"
	}
	return mode
}

// installationModeFromAPI is read-side: keep legacy SCAN_ALL visible so state
// matches the API. Only an absent mode gets the schema default.
func installationModeFromAPI(mode string) string {
	if mode == "" {
		return "SELECTED_REPOSITORIES"
	}
	return mode
}

func ExpandUpdate(mode types.String, defaultPolicies types.Bool, policiesIds types.Set, cfg *ConfigSettingsModel) api_client.ScmInstallationUpdate {
	ids := tfconv.SetToStringSlice(policiesIds)
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

// Plan values are unreliable for project intent because UseStateForUnknown backfills omitted fields.
type ProjectIntent struct {
	FromConfig     types.String
	PoliciesIntent bool
}

// hasExplicitPolicies: only explicit policies_ids is mutually exclusive with project_id.
func hasExplicitPolicies(policies types.Set) bool {
	return tfconv.Known(policies)
}

func ProjectIntentFrom(configProjectID types.String, configPolicies types.Set) ProjectIntent {
	return ProjectIntent{
		FromConfig:     configProjectID,
		PoliciesIntent: hasExplicitPolicies(configPolicies),
	}
}

// Defaults for create and empty enum fallback. DisableScanPullRequests=false overrides GitHub backend default (true).
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

// CreateUnitBody is Adopt seeded with defaults for a brand-new unit.
func CreateUnitBody(mode types.String, planDefault types.Bool, planPolicies types.Set, planConfig *ConfigSettingsModel, project ProjectIntent) api_client.ScmInstallationUpdate {
	seed := api_client.ScmUnitCommonFields{
		InstallationMode: mode.ValueString(),
		DefaultPolicies:  !project.PoliciesIntent,
		ConfigSettings:   defaultConfigSettings(),
	}
	return Adopt(mode, planDefault, planPolicies, planConfig, project, seed)
}

// Adopt hydrates unset plan fields from the live unit (seed on create).
// PUT sends project_id XOR explicit policies; default_policies may accompany project_id.
func Adopt(planMode types.String, planDefault types.Bool, planPolicies types.Set, planConfig *ConfigSettingsModel, project ProjectIntent, ex api_client.ScmUnitCommonFields) api_client.ScmInstallationUpdate {
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
		policies = PolicyIDsFromRefs(ex.Policies)
	}

	projectID := api_client.ProjectRefID(ex.Project)
	switch {
	case !project.FromConfig.IsNull() && !project.FromConfig.IsUnknown():
		projectID = project.FromConfig.ValueString() // explicit project_id wins (may be "" to clear)
	case project.PoliciesIntent:
		projectID = ""
	}
	// Explicit policies_ids always means default_policies=false on the wire.
	// Plan may still carry default_policies=true via UseStateForUnknown when the
	// attribute was omitted; ExpandUpdate would otherwise discard the ids.
	if project.PoliciesIntent {
		defaultPolicies = types.BoolValue(false)
	}

	body := ExpandUpdate(mode, defaultPolicies, policies, &merged)
	if projectID != "" {
		body.ProjectID = projectID
		body.Policies = nil
	}
	return body
}
