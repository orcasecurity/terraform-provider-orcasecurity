package shift_left_azure_devops_account

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func resourceSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes("Azure DevOps account/organization name.")
	// SharedScmConfigAttributes already defines computed account_name — override to Required identity.
	attrs["account_name"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Azure DevOps organization name (API `azure_account_name` on integrate).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["id"] = rschema.StringAttribute{
		Computed:      true,
		Description:   "Orca Azure DevOps integrated account UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca Azure DevOps installation UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Creates or configures an Orca Azure DevOps shift-left integrated account (organization). " +
			"Create POSTs `/api/shiftleft/azure_devops/integrated_repositories/` with `azure_account_name`, " +
			"`installation_mode` (defaults to `SCAN_ALL_INCLUDE_FUTURE`), configuration, and empty `repositories`. " +
			"If already integrated, Create/Update PUT the unit config. Destroy DELETEs the integrated account. " +
			"Not covered: browse accounts, check_availability, scan-now. " +
			"Schema follows the Shift-Left API (a superset of the Azure UI, which hides skip_check_runs and archive actions).",
		Attributes: attrs,
	}
}
