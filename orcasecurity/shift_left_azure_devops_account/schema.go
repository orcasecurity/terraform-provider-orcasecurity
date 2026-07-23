package shift_left_azure_devops_account

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func resourceSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes("Azure DevOps account/organization name.")
	attrs["id"] = rschema.StringAttribute{
		Computed:      true,
		Description:   "Account UUID (mirrors account_id).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca Azure DevOps installation UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["account_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca Azure DevOps integrated account UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Configures an existing Orca Azure DevOps shift-left integrated account (default policies, scan mode, PR settings). The account must already be integrated. Adopt via `terraform import`. Schema follows the Shift-Left API (a superset of the Azure UI, which hides skip_check_runs and archive actions).",
		Attributes:  attrs,
	}
}
