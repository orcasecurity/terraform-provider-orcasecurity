package shift_left_github_installation

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func resourceSchema() rschema.Schema {
	return rschema.Schema{
		Description: "Configures an existing Orca GitHub shift-left installation (default policies, scan mode, PR settings). The installation must already exist (created by installing the Orca GitHub App). Adopt via `terraform import`.",
		Attributes: map[string]rschema.Attribute{
			"id": rschema.StringAttribute{
				Computed:      true,
				Description:   "Installation UUID (mirrors installation_id).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"installation_id": rschema.StringAttribute{
				Required:      true,
				Description:   "Orca installation UUID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"account_name": rschema.StringAttribute{
				Computed:    true,
				Description: "GitHub account/organization name.",
			},
			"installation_mode": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Scan mode, e.g. SCAN_ALL_INCLUDE_FUTURE.",
			},
			"default_policies": rschema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Attach all Orca built-in policies. When true, policies_ids is ignored.",
			},
			"policies_ids": rschema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Explicit policy IDs to attach (used when default_policies is false).",
			},
			"configuration_settings": rschema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "PR/MR advanced settings.",
				Attributes:  shift_left_integration.ConfigSettingsAttributes(shift_left_integration.FieldGate{ArchiveActions: true}),
			},
		},
	}
}
