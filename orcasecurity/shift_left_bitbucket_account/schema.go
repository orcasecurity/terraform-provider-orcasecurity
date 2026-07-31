package shift_left_bitbucket_account

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func resourceSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes("Bitbucket workspace/account name.")
	// The backend has no Bitbucket scope on SCM posture policies, so the shared
	// attribute can never carry a value here.
	attrs["scm_posture_policy_id"] = shift_left_integration.ComputedVolatileString(
		"Always null for Bitbucket: SCM posture policies cannot scope Bitbucket workspaces, so no posture policy ever attaches to this unit. Present for schema parity with the other SCM account resources.",
	)
	attrs["id"] = rschema.StringAttribute{
		Computed:      true,
		Description:   "Orca Bitbucket integrated account UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca Bitbucket installation UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["account_id"] = rschema.StringAttribute{
		Required: true,
		Description: "Bitbucket-side workspace slug (cloud) or project key (server). " +
			"This is NOT an Orca UUID — the Orca unit id is the computed `id`. " +
			"Do not confuse with `orcasecurity_shift_left_bitbucket_installation.account_id` " +
			"(token-scope slug on the installation credential).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Creates or configures an Orca Bitbucket shift-left integrated account. " +
			"Create POSTs `/api/shiftleft/bitbucket/integrated_repositories/` with Bitbucket `account_id` (slug), " +
			"`installation_mode`, and configuration (no repositories are attached on that call). " +
			"Integrating a not-yet-integrated workspace requires `installation_mode = \"SCAN_ALL_INCLUDE_FUTURE\"`: " +
			"the API accepts `SELECTED_REPOSITORIES` on integrate only together with an explicit repository list, " +
			"which this resource does not send (you can switch modes on a later apply). " +
			"If already integrated, Create/Update PUT the unit config. Destroy DELETEs the integrated account. " +
			"Not covered: browse accounts, check_availability, scan-now. " +
			"Archive/unavailable actions in configuration_settings may be ignored by the Bitbucket API.",
		Attributes: attrs,
	}
}
