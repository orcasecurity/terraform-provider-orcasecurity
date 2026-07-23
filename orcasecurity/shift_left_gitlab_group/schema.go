package shift_left_gitlab_group

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func resourceSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes("GitLab group/account name.")
	attrs["id"] = rschema.StringAttribute{
		Computed:      true,
		Description:   "Orca GitLab integrated group UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca GitLab installation UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["gitlab_group_id"] = rschema.Int64Attribute{
		Required:      true,
		Description:   "GitLab-side numeric group ID (from GET .../installations/{id}/groups/).",
		PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Creates or configures an Orca GitLab shift-left integrated group. " +
			"Create POSTs `/api/shiftleft/gitlab/integrated_repositories/` with `group_id`, " +
			"`installation_mode` (defaults to `SCAN_ALL_INCLUDE_FUTURE`), configuration, and empty `repositories` " +
			"(UI parity). If the group is already integrated, Create/Update PUT the unit config instead. " +
			"Destroy DELETEs the integrated group (tears down the live integration and its repos). " +
			"Not covered: browse remote groups, check_availability, scan-now (UI operations).",
		Attributes: attrs,
	}
}
