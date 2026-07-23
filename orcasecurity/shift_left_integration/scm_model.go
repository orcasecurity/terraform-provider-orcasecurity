package shift_left_integration

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ScmConfigFields are the shared Terraform attributes every SCM adopt-existing
// resource carries (beyond provider-specific identity keys).
type ScmConfigFields struct {
	AccountName       types.String         `tfsdk:"account_name"`
	IntegrationStatus types.String         `tfsdk:"integration_status"`
	InstallationMode  types.String         `tfsdk:"installation_mode"`
	DefaultPolicies   types.Bool           `tfsdk:"default_policies"`
	PoliciesIds       types.Set            `tfsdk:"policies_ids"`
	ProjectID         types.String         `tfsdk:"project_id"`
	ConfigSettings    *ConfigSettingsModel `tfsdk:"configuration_settings"`

	// Read-only status fields surfaced from the API.
	ScanAllState                types.String `tfsdk:"scan_all_state"`
	IntegratedRepositoriesCount types.Int64  `tfsdk:"integrated_repositories_count"`
	ScmPosturePolicyID          types.String `tfsdk:"scm_posture_policy_id"`
}

// OptionalID maps an API id into state: empty becomes null so Optional+Computed
// attributes do not drift between null (unset) and "".
func OptionalID(id string) types.String {
	if id == "" {
		return types.StringNull()
	}
	return types.StringValue(id)
}

// ScmConfigFieldsFromAPI builds the shared config fields from a live SCM unit.
func ScmConfigFieldsFromAPI(accountName string, u api_client.ScmUnitCommonFields) ScmConfigFields {
	cs := FlattenConfigSettings(u.ConfigSettings)
	return ScmConfigFields{
		AccountName:       types.StringValue(accountName),
		IntegrationStatus: OptionalID(u.IntegrationStatus),
		InstallationMode:  types.StringValue(u.InstallationMode),
		DefaultPolicies:   types.BoolValue(u.DefaultPolicies),
		PoliciesIds:       PolicyIDsFromRefs(u.Policies),
		ProjectID:         OptionalID(api_client.ProjectRefID(u.Project)),
		ConfigSettings:    &cs,

		ScanAllState:                OptionalID(u.ScanAllState),
		IntegratedRepositoriesCount: types.Int64Value(u.IntegratedRepositoriesCount),
		ScmPosturePolicyID:          OptionalID(u.ScmPosturePolicyID),
	}
}
