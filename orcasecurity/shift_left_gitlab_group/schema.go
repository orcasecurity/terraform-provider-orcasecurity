package shift_left_gitlab_group

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func resourceSchema() rschema.Schema {
	return rschema.Schema{
		Description: "Configures an existing Orca GitLab shift-left integrated group (default policies, scan mode, PR/MR settings). The group must already be integrated (created by installing the Orca GitLab integration). Adopt via `terraform import`.",
		Attributes: map[string]rschema.Attribute{
			"id": rschema.StringAttribute{
				Computed:      true,
				Description:   "Group UUID (mirrors group_id).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"installation_id": rschema.StringAttribute{
				Required:      true,
				Description:   "Orca GitLab installation UUID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"group_id": rschema.StringAttribute{
				Required:      true,
				Description:   "Orca GitLab integrated group UUID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"account_name": rschema.StringAttribute{
				Computed:    true,
				Description: "GitLab group/account name.",
			},
			"installation_mode": rschema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Scan mode: SCAN_ALL_INCLUDE_FUTURE or SELECTED_REPOSITORIES.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.OneOf("SCAN_ALL_INCLUDE_FUTURE", "SELECTED_REPOSITORIES"),
				},
			},
			"default_policies": rschema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Attach all Orca built-in policies. When true, policies_ids is ignored. Mutually exclusive with project_id.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"policies_ids": rschema.SetAttribute{
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				Description:   "Explicit policy IDs to attach (used when default_policies is false). Mutually exclusive with project_id.",
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"project_id": rschema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Bind this unit to a scan-all project instead of policies. Mutually exclusive with policies_ids and default_policies. Set to an empty string to clear the binding; omit to leave it unchanged.",
				PlanModifiers: []planmodifier.String{shift_left_integration.ProjectIDPlanModifier()},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("policies_ids"), path.MatchRoot("default_policies")),
				},
			},
			"configuration_settings": rschema.SingleNestedAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "PR/MR advanced settings.",
				Attributes:    shift_left_integration.ConfigSettingsAttributes(),
				PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
			},
		},
	}
}
