package shift_left_gitlab_group

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func resourceSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes("GitLab group/account name.")
	attrs["id"] = rschema.StringAttribute{
		Computed:      true,
		Description:   "Group UUID (mirrors group_id).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca GitLab installation UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["group_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca GitLab integrated group UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Configures an existing Orca GitLab shift-left integrated group (default policies, scan mode, PR/MR settings). The group must already be integrated (created by installing the Orca GitLab integration). Adopt via `terraform import`.",
		Attributes:  attrs,
	}
}
