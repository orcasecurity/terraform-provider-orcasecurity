package shift_left_integration

import (
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

// SharedScmConfigAttributes returns the configuration attributes common to every
// SCM adopt-existing resource (GitHub/GitLab/Azure/Bitbucket). Callers merge
// these with their provider-specific identity attributes (installation_id,
// group_id/account_id, etc.).
//
// Schema follows the Shift-Left API contract (not the stricter per-provider UI
// feature gates). Azure's UI hides skip_check_runs and archive actions, and
// GitLab's UI often limits skip_check_runs to ALWAYS/NEVER, but the account-
// level PUT accepts the full enums for all providers.
func SharedScmConfigAttributes(accountNameDescription string) map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
		"account_name": rschema.StringAttribute{
			Computed:    true,
			Description: accountNameDescription,
		},
		"integration_status": rschema.StringAttribute{
			Computed:    true,
			Description: "Live integration health from the API (e.g. ENABLED, DISABLED_DUE_TO_INVALID_TOKEN, INSTALLATION_SUSPENDED, INSTALLATION_UNREACHABLE). Null when the API omits it.",
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
			PlanModifiers: []planmodifier.String{ProjectIDPlanModifier()},
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.MatchRoot("policies_ids"), path.MatchRoot("default_policies")),
			},
		},
		"configuration_settings": rschema.SingleNestedAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "PR/MR advanced settings. Follows the API surface (full skip_check_runs and archive/unavailable enums for every provider), which is a superset of what some SCM UIs expose.",
			Attributes:    ConfigSettingsAttributes(),
			PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
		},
		"scan_all_state": rschema.StringAttribute{
			Computed:    true,
			Description: "Read-only state of the scan-all onboarding flow for this unit (null when the API omits it).",
		},
		"integrated_repositories_count": rschema.Int64Attribute{
			Computed:    true,
			Description: "Read-only count of repositories integrated under this unit.",
		},
		"scm_posture_policy_id": rschema.StringAttribute{
			Computed:    true,
			Description: "Read-only ID of the SCM posture policy attached to this unit (null when none).",
		},
	}
}
