package shift_left_bitbucket_account

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func resourceSchema() rschema.Schema {
	return rschema.Schema{
		Description: "Configures an existing Orca Bitbucket shift-left integrated account (default policies, scan mode, PR settings). The account must already be integrated (created by installing the Orca Bitbucket integration). Adopt via `terraform import`. Archive/unavailable repository actions are not supported for Bitbucket.",
		Attributes: map[string]rschema.Attribute{
			"id": rschema.StringAttribute{
				Computed:      true,
				Description:   "Account UUID (mirrors account_id).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"installation_id": rschema.StringAttribute{
				Required:      true,
				Description:   "Orca Bitbucket installation UUID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"account_id": rschema.StringAttribute{
				Required:      true,
				Description:   "Orca Bitbucket integrated account UUID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"account_name": rschema.StringAttribute{
				Computed:    true,
				Description: "Bitbucket account/workspace name.",
			},
			"installation_mode": rschema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Scan mode, e.g. SCAN_ALL_INCLUDE_FUTURE.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"default_policies": rschema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Attach all Orca built-in policies. When true, policies_ids is ignored.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"policies_ids": rschema.SetAttribute{
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				Description:   "Explicit policy IDs to attach (used when default_policies is false).",
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"configuration_settings": rschema.SingleNestedAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "PR advanced settings.",
				Attributes:    shift_left_integration.ConfigSettingsAttributes(shift_left_integration.FieldGate{ArchiveActions: false}),
				PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
			},
		},
	}
}
