package shift_left_github_installation

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resourceModel struct {
	ID               types.String                                `tfsdk:"id"`
	InstallationID   types.String                                `tfsdk:"installation_id"`
	AccountName      types.String                                `tfsdk:"account_name"`
	InstallationMode types.String                                `tfsdk:"installation_mode"`
	DefaultPolicies  types.Bool                                  `tfsdk:"default_policies"`
	PoliciesIds      types.Set                                   `tfsdk:"policies_ids"`
	ProjectID        types.String                                `tfsdk:"project_id"`
	ConfigSettings   *shift_left_integration.ConfigSettingsModel `tfsdk:"configuration_settings"`
}
