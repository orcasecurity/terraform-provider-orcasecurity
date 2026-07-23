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
func ScmConfigFieldsFromAPI(
	accountName string,
	integrationStatus string,
	mode string,
	defaultPolicies bool,
	policies []api_client.ScmPolicyRef,
	project *api_client.ScmProjectRef,
	cfg api_client.ShiftLeftConfigSettings,
) ScmConfigFields {
	cs := FlattenConfigSettings(cfg)
	return ScmConfigFields{
		AccountName:       types.StringValue(accountName),
		IntegrationStatus: OptionalID(integrationStatus),
		InstallationMode:  types.StringValue(mode),
		DefaultPolicies:   types.BoolValue(defaultPolicies),
		PoliciesIds:       PolicyIDsFromRefs(policies),
		ProjectID:         OptionalID(api_client.ProjectRefID(project)),
		ConfigSettings:    &cs,
	}
}
