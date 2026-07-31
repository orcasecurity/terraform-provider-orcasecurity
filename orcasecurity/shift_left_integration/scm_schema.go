package shift_left_integration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nonEmptyPoliciesValidator: server treats empty/omitted policies as "attach all built-ins", so [] != none.
type nonEmptyPoliciesValidator struct{}

func (nonEmptyPoliciesValidator) Description(_ context.Context) string {
	return "policies_ids must not be an empty set"
}

func (v nonEmptyPoliciesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (nonEmptyPoliciesValidator) ValidateSet(_ context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if len(req.ConfigValue.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Empty policies_ids",
			"policies_ids = [] does not attach zero policies. The API treats an empty policies list the same as "+
				"attaching all Orca built-in policies (equivalent to default_policies = true). To attach specific "+
				"policies, list their IDs; to attach all built-ins, set default_policies = true. Attaching zero "+
				"policies is not supported by the API.",
		)
	}
}

func SharedScmConfigAttributes(accountNameDescription string) map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
		"account_name": rschema.StringAttribute{
			Computed:      true,
			Description:   accountNameDescription,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"integration_status": ComputedVolatileString(
			"Live integration health from the API (e.g. ENABLED, DISABLED_DUE_TO_INVALID_TOKEN, INSTALLATION_SUSPENDED, INSTALLATION_UNREACHABLE). Null when the API omits it. Volatile: re-read after writes; settled across no-op plans.",
		),
		"installation_mode": rschema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "Scan mode: SCAN_ALL_INCLUDE_FUTURE or SELECTED_REPOSITORIES. Defaults to SELECTED_REPOSITORIES when omitted (matches the API/UI); SCAN_ALL_INCLUDE_FUTURE enrolls every current and future repository for scanning.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Validators: []validator.String{
				stringvalidator.OneOf("SCAN_ALL_INCLUDE_FUTURE", "SELECTED_REPOSITORIES"),
			},
		},
		"default_policies": rschema.BoolAttribute{
			Optional: true,
			Computed: true,
			Description: "On write, `true` attaches every Orca built-in policy (and clears any explicit `policies_ids`). " +
				"On read the API derives this flag as true only when the unit has neither attached policies nor a project — " +
				"it is not a stored boolean. Mutually exclusive with `policies_ids`; may accompany `project_id`. " +
				"`false` alone (no `policies_ids`, no `project_id`) can never round-trip.",
			PlanModifiers: []planmodifier.Bool{DefaultPoliciesPlanModifier()},
		},
		"policies_ids": rschema.SetAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: "Explicit policy IDs to attach. Mutually exclusive with `project_id` and with `default_policies = true`. " +
				"Must be non-empty: an empty list is treated by the API as \"attach all Orca built-in policies\", not \"none\".",
			PlanModifiers: []planmodifier.Set{PoliciesIDsPlanModifier()},
			Validators:    []validator.Set{nonEmptyPoliciesValidator{}},
		},
		"project_id": rschema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "Bind this unit to a scan-all project. Mutually exclusive with policies_ids (the API rejects both); default_policies may still apply. Set to an empty string to clear the binding; omit to leave it unchanged.",
			PlanModifiers: []planmodifier.String{ProjectIDPlanModifier()},
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.MatchRoot("policies_ids")),
			},
		},
		// Input-only: never returned by the API, so it must not be Computed. Marking it
		// Computed makes it plan as unknown on create, and the unknown then reaches state.
		"adopt_existing": rschema.BoolAttribute{
			Optional:    true,
			Description: "Acknowledge takeover of a unit that is already integrated in Orca. These resources ADOPT a pre-existing SCM unit rather than create one: applying takes over the live integration, and a later destroy DE-INTEGRATES it (removing repositories and settings that may have been configured outside Terraform). As a guard, Create refuses to silently take over a unit that already has integrated repositories unless this is set to true. Prefer `terraform import` to bring an existing unit under management without a takeover write; set this to true only when you intend to manage (and eventually tear down) an integration you did not create here.",
		},
		"configuration_settings": rschema.SingleNestedAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "PR/MR advanced settings. Follows the API surface (full skip_check_runs and archive/unavailable enums for every provider), which is a superset of what some SCM UIs expose.",
			Attributes:    ConfigSettingsAttributes(),
			PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
		},
		"scan_all_state": ComputedVolatileString(
			"Read-only state of the scan-all onboarding flow for this unit (null when the API omits it). Volatile: re-read after writes; settled across no-op plans.",
		),
		"integrated_repositories_count": ComputedVolatileInt64(
			"Read-only count of repositories integrated under this unit. Volatile: may change as a side effect of installation_mode changes (e.g. SCAN_ALL_INCLUDE_FUTURE enrollment); settled across no-op plans.",
		),
		"scm_posture_policy_id": ComputedVolatileString(
			"Read-only ID of the SCM posture policy attached to this unit (null when none). Volatile: re-read after writes; settled across no-op plans.",
		),
	}
}
