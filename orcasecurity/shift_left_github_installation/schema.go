package shift_left_github_installation

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func resourceSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes("GitHub account/organization name.")
	attrs["id"] = rschema.StringAttribute{
		Computed:      true,
		Description:   "Installation UUID (mirrors installation_id).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca installation UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["github_installation_id"] = rschema.Int64Attribute{
		Computed:    true,
		Description: "GitHub-side numeric installation ID of the Orca GitHub App.",
	}
	attrs["github_app_settings_url"] = rschema.StringAttribute{
		Computed:    true,
		Description: "URL of the Orca GitHub App settings page on GitHub (null when the API omits it).",
	}
	return rschema.Schema{
		Description: "Configures an existing Orca GitHub shift-left installation (default policies, scan mode, PR settings). The installation must already exist (created by installing the Orca GitHub App). Adopt via `terraform import`. Schema follows the Shift-Left API (a superset of the UI): all `configuration_settings` enums are available.",
		Attributes:  attrs,
	}
}
