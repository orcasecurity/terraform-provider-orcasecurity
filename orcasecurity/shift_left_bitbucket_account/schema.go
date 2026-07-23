package shift_left_bitbucket_account

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func resourceSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes("Bitbucket workspace/account name.")
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
		Required:      true,
		Description:   "Bitbucket-side account/workspace slug (API `account_id`, not the Orca UUID).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Creates or configures an Orca Bitbucket shift-left integrated account. " +
			"Create POSTs `/api/shiftleft/bitbucket/integrated_repositories/` with Bitbucket `account_id` (slug), " +
			"`installation_mode` (defaults to `SCAN_ALL_INCLUDE_FUTURE`), configuration, and empty `repositories`. " +
			"If already integrated, Create/Update PUT the unit config. Destroy DELETEs the integrated account. " +
			"Not covered: browse accounts, check_availability, scan-now. " +
			"Archive/unavailable actions in configuration_settings may be ignored by the Bitbucket API.",
		Attributes: attrs,
	}
}
