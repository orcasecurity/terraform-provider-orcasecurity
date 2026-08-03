package shift_left_integration

import (
	"errors"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

var ErrUnitNotFound = errors.New("scm unit not found")

// Commoner is satisfied by every concrete SCM unit API type via the embedded
// api_client.ScmUnitCommonFields.
type Commoner interface {
	Common() api_client.ScmUnitCommonFields
}

func PolicyIDsFromRefs(refs []api_client.ScmPolicyRef) types.Set {
	return tfconv.StringSliceToSet(api_client.PolicyRefIDs(refs))
}

type ScmConfigFields struct {
	AccountName       types.String `tfsdk:"account_name"`
	IntegrationStatus types.String `tfsdk:"integration_status"`
	InstallationMode  types.String `tfsdk:"installation_mode"`
	DefaultPolicies   types.Bool   `tfsdk:"default_policies"`
	PoliciesIds       types.Set    `tfsdk:"policies_ids"`
	ProjectID         types.String `tfsdk:"project_id"`
	AdoptExisting     types.Bool   `tfsdk:"adopt_existing"`
	// types.Object rather than *ConfigSettingsModel: the attribute is Optional+Computed and
	// so plans as unknown on create, which a Go struct pointer cannot represent.
	ConfigSettings types.Object `tfsdk:"configuration_settings"`

	ScanAllState                types.String `tfsdk:"scan_all_state"`
	IntegratedRepositoriesCount types.Int64  `tfsdk:"integrated_repositories_count"`
	ScmPosturePolicyID          types.String `tfsdk:"scm_posture_policy_id"`
}

func ScmConfigFieldsFromAPI(accountName string, u api_client.ScmUnitCommonFields) ScmConfigFields {
	cs := FlattenConfigSettings(u.ConfigSettings)
	return ScmConfigFields{
		AccountName:       types.StringValue(accountName),
		IntegrationStatus: tfconv.StringOrNull(u.IntegrationStatus),
		// Report the API value as-is. Write paths still remap legacy SCAN_ALL via
		// normalizeInstallationMode; rewriting on read hid units that are not in
		// SELECTED_REPOSITORIES and removed the drift signal.
		InstallationMode: types.StringValue(installationModeFromAPI(u.InstallationMode)),
		DefaultPolicies:  types.BoolValue(u.DefaultPolicies),
		PoliciesIds:      PolicyIDsFromRefs(u.Policies),
		ProjectID:        tfconv.StringOrNull(api_client.ProjectRefID(u.Project)),
		ConfigSettings:   ConfigSettingsToObject(cs),

		ScanAllState:                tfconv.StringOrNull(u.ScanAllState),
		IntegratedRepositoriesCount: types.Int64Value(u.IntegratedRepositoriesCount),
		ScmPosturePolicyID:          tfconv.StringOrNull(u.ScmPosturePolicyID),
	}
}

// preserveKnownEmpties reconciles the values the API cannot round-trip. Clearing a project
// binding, a PR summary appendix, or a condition list makes the API report the field as
// absent, which maps back to null — but Terraform requires an Optional+Computed attribute
// to end up equal to a non-null configured value, and "", [] are the documented ways to
// clear these. Applied on the write path (prior = plan) and the read path (prior = prior
// state), so a cleared value neither fails the apply nor reappears as drift on refresh.
func preserveKnownEmpties(next, prior *ScmConfigFields) {
	if next.ProjectID.IsNull() && isKnownEmptyString(prior.ProjectID) {
		next.ProjectID = types.StringValue("")
	}

	nextCfg := ConfigSettingsFromObject(next.ConfigSettings)
	priorCfg := ConfigSettingsFromObject(prior.ConfigSettings)
	if nextCfg == nil || priorCfg == nil {
		return
	}
	if nextCfg.PrSummaryAppendix.IsNull() && isKnownEmptyString(priorCfg.PrSummaryAppendix) {
		nextCfg.PrSummaryAppendix = types.StringValue("")
	}
	preserveEmptyList(&nextCfg.ArchiveConditions, priorCfg.ArchiveConditions)
	preserveEmptyList(&nextCfg.UnavailableConditions, priorCfg.UnavailableConditions)
	next.ConfigSettings = ConfigSettingsToObject(*nextCfg)
}

func isKnownEmptyString(v types.String) bool {
	return tfconv.Known(v) && v.ValueString() == ""
}

func preserveEmptyList(next *types.List, prior types.List) {
	if next.IsNull() && tfconv.Known(prior) && len(prior.Elements()) == 0 {
		*next = types.ListValueMust(types.StringType, nil)
	}
}
