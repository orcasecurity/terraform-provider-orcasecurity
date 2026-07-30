package shift_left_github_account

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resourceModel struct {
	ID types.String `tfsdk:"id"`
	// AccountID is the Orca UUID of the integrated GitHub account. Unlike the other SCMs
	// there is no parent connection resource — the Orca GitHub App installation is itself
	// the account-level unit — so this is the unit's own id and `id` mirrors it.
	AccountID            types.String `tfsdk:"account_id"`
	GithubInstallationID types.Int64  `tfsdk:"github_installation_id"`
	GithubAppSettingsURL types.String `tfsdk:"github_app_settings_url"`
	shift_left_integration.ScmConfigFields
}
