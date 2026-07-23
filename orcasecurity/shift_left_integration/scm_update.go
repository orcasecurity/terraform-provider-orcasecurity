package shift_left_integration

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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

// The API still returns SCAN_ALL on old units but rejects it on update.
func normalizeInstallationMode(mode string) string {
	if mode == "SCAN_ALL" {
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
		InstallationMode: mode.ValueString(),
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
}

// Plan values are unreliable for project intent because UseStateForUnknown backfills omitted fields.
type ProjectIntent struct {
	FromConfig     types.String
	PoliciesIntent bool
}

func policiesIntent(policies types.Set, defaultPolicies types.Bool) bool {
	return (!policies.IsNull() && !policies.IsUnknown()) ||
		(!defaultPolicies.IsNull() && !defaultPolicies.IsUnknown())
}

func ProjectIntentFrom(configProjectID types.String, configPolicies types.Set, configDefault types.Bool) ProjectIntent {
	return ProjectIntent{
		FromConfig:     configProjectID,
		PoliciesIntent: policiesIntent(configPolicies, configDefault),
	}
}

type Adopted struct {
	InstallationMode types.String
	DefaultPolicies  types.Bool
	PoliciesIds      types.Set
	ConfigSettings   *ConfigSettingsModel
	Body             api_client.ScmInstallationUpdate
}

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

// Adopt hydrates unset plan fields from the live unit so apply never wipes server-managed state.
// The PUT sends project_id XOR policies, matching the UI.
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

	projectID := ex.ProjectID
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
