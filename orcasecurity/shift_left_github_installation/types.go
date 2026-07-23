package shift_left_github_installation

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resourceModel struct {
	ID                   types.String `tfsdk:"id"`
	InstallationID       types.String `tfsdk:"installation_id"`
	GithubInstallationID types.Int64  `tfsdk:"github_installation_id"`
	GithubAppSettingsURL types.String `tfsdk:"github_app_settings_url"`
	shift_left_integration.ScmConfigFields
}
