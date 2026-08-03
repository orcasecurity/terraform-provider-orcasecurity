package shift_left_github_account

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func resourceSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes()
	attrs["account_name"] = shift_left_integration.ComputedAccountName("GitHub account/organization name.")
	attrs["id"] = rschema.StringAttribute{
		Computed: true,
		Description: "Orca UUID of the integrated GitHub account (same value as `account_id`). " +
			"Unlike Bitbucket, GitHub `account_id` is an Orca UUID, not an SCM-side slug.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["account_id"] = rschema.StringAttribute{
		Required: true,
		Description: "Orca UUID of the integrated GitHub account (see `orcasecurity_shift_left_github_accounts`). " +
			"Unlike Bitbucket's `account_id` (an SCM slug), this is an Orca UUID. " +
			"GitHub has no separate installation resource: the Orca GitHub App installation is itself the " +
			"account-level unit, so this is the unit's own id rather than a parent connection id.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["github_installation_id"] = shift_left_common.ComputedInt64(
		"GitHub-side numeric installation ID of the Orca GitHub App.")
	attrs["github_app_settings_url"] = shift_left_common.ComputedString(
		"URL of the Orca GitHub App settings page on GitHub (null when the API omits it).")
	return rschema.Schema{
		Description: "Configures an existing Orca GitHub shift-left account/organization (default policies, scan mode, PR settings). " +
			"This is the GitHub peer of `orcasecurity_shift_left_gitlab_group` and `orcasecurity_shift_left_azure_devops_account`. " +
			"The account must already exist (created by installing the Orca GitHub App — `/github/config/` and App callback are not managed here). " +
			"Create/Update PUT the unit config; Destroy DELETEs the Orca integration (tears down the live integration). " +
			"Not covered: GHES `/github/enterprises/*`, browse repos, check_availability, scan-now. " +
			"Schema follows the Shift-Left API (a superset of the UI): all `configuration_settings` enums are available.",
		Attributes: attrs,
	}
}
