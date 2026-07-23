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
		Description:   "Account UUID (mirrors account_id).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca Bitbucket installation UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["account_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca Bitbucket integrated account UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Configures an existing Orca Bitbucket shift-left integrated account (default policies, scan mode, PR settings). The account must already be integrated (created by installing the Orca Bitbucket integration). Adopt via `terraform import`. Archive/unavailable repository actions are accepted in configuration_settings but may be ignored by the Bitbucket API.",
		Attributes:  attrs,
	}
}
