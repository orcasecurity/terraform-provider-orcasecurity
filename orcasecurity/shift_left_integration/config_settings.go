package shift_left_integration

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type ConfigSettingsModel struct {
	shift_left_common.PRSettingsModel
	PrSummaryAppendix     types.String `tfsdk:"pr_summary_appendix"`
	ArchiveConditions     types.List   `tfsdk:"archive_conditions"`
	UnavailableConditions types.List   `tfsdk:"unavailable_conditions"`
}

func ConfigSettingsAttributes() map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"disable_scan_pull_requests": schema.BoolAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "Disable scanning pull requests.",
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		},
		"comments_on_pull_requests": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "When to post scan result comments on pull requests.",
			Validators:    shift_left_common.PRCommentValidator(),
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"pr_summary_comment": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "When to post a pull request summary comment.",
			Validators:    shift_left_common.PRCommentValidator(),
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"skip_check_runs": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "When to skip posting check runs.",
			Validators:    shift_left_common.SkipCheckRunsValidator(shift_left_common.FullSkipCheckRunValues),
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"config_file_support": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "Whether in-repo Orca config file support is enabled.",
			Validators:    shift_left_common.ConfigFileSupportValidator(),
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"pr_summary_appendix": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "Additional free-text appendix appended to the pull request summary comment.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}

	attrs["archive_conditions"] = schema.ListAttribute{
		Optional:    true,
		Computed:    true,
		ElementType: types.StringType,
		Description: "Conditions that trigger an archive action for repositories (installation_repositories_configuration.archive_actions.conditions). API accepts AVOID_SCAN and DELETE_REPO.",
		Validators: []validator.List{
			listvalidator.ValueStringsAre(stringvalidator.OneOf("AVOID_SCAN", "DELETE_REPO")),
		},
		PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
	}
	attrs["unavailable_conditions"] = schema.ListAttribute{
		Optional:    true,
		Computed:    true,
		ElementType: types.StringType,
		Description: "Conditions that trigger an action when a repository becomes unavailable (installation_repositories_configuration.unavailable_actions.conditions). API accepts AVOID_SCAN and DELETE_REPO (same as archive_conditions).",
		Validators: []validator.List{
			listvalidator.ValueStringsAre(stringvalidator.OneOf("AVOID_SCAN", "DELETE_REPO")),
		},
		PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
	}

	return attrs
}

// ConfigSettingsAttrTypes mirrors ConfigSettingsAttributes. configuration_settings is
// Optional+Computed, so it plans as unknown on create; the model therefore travels as a
// types.Object (which can hold unknown) and is decoded through ConfigSettingsFromObject.
func ConfigSettingsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"disable_scan_pull_requests": types.BoolType,
		"comments_on_pull_requests":  types.StringType,
		"pr_summary_comment":         types.StringType,
		"skip_check_runs":            types.StringType,
		"config_file_support":        types.StringType,
		"pr_summary_appendix":        types.StringType,
		"archive_conditions":         types.ListType{ElemType: types.StringType},
		"unavailable_conditions":     types.ListType{ElemType: types.StringType},
	}
}

func ConfigSettingsObjectNull() types.Object {
	return types.ObjectNull(ConfigSettingsAttrTypes())
}

// ConfigSettingsFromObject decodes the nested block. A null or unknown object yields
// nil, which Adopt reads as "inherit every field from the live unit". The object
// shape is provider-owned (ConfigSettingsAttrTypes), so a decode failure cannot
// happen outside a programming error and maps to nil as well.
func ConfigSettingsFromObject(ctx context.Context, obj types.Object) *ConfigSettingsModel {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	var m ConfigSettingsModel
	if diags := obj.As(ctx, &m, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil
	}
	return &m
}

func ConfigSettingsToObject(m ConfigSettingsModel) types.Object {
	return types.ObjectValueMust(ConfigSettingsAttrTypes(), map[string]attr.Value{
		"disable_scan_pull_requests": m.DisableScanPullRequests,
		"comments_on_pull_requests":  m.CommentsOnPullRequests,
		"pr_summary_comment":         m.PrSummaryComment,
		"skip_check_runs":            m.SkipCheckRuns,
		"config_file_support":        m.ConfigFileSupport,
		"pr_summary_appendix":        m.PrSummaryAppendix,
		"archive_conditions":         m.ArchiveConditions,
		"unavailable_conditions":     m.UnavailableConditions,
	})
}

func stringSliceFromList(l types.List) []string {
	if l.IsNull() || l.IsUnknown() {
		return nil
	}
	elements := l.Elements()
	if len(elements) == 0 {
		return nil
	}
	result := make([]string, 0, len(elements))
	for _, e := range elements {
		s, ok := e.(types.String)
		if !ok || s.IsNull() || s.IsUnknown() {
			continue
		}
		result = append(result, s.ValueString())
	}
	return result
}

func stringSliceToList(values []string) types.List {
	if len(values) == 0 {
		return types.ListNull(types.StringType)
	}
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}

// Required enums default when unset — legacy units return "" and PATCH rejects empty. pr_summary_appendix is not defaulted ("" clears it).
func ExpandConfigSettings(m *ConfigSettingsModel) api_client.ShiftLeftConfigSettings {
	out := defaultConfigSettings()
	if m == nil {
		return out
	}

	out.DisableScanPullRequests = m.DisableScanPullRequests.ValueBool()
	out.PrSummaryAppendix = m.PrSummaryAppendix.ValueString()
	setEnum := func(dst *string, src types.String) {
		if v := src.ValueString(); v != "" {
			*dst = v
		}
	}
	setEnum(&out.CommentsOnPullRequests, m.CommentsOnPullRequests)
	setEnum(&out.PrSummaryComment, m.PrSummaryComment)
	setEnum(&out.SkipCheckRuns, m.SkipCheckRuns)
	setEnum(&out.ConfigFileSupport, m.ConfigFileSupport)

	archiveKnown := tfconv.Known(m.ArchiveConditions)
	unavailableKnown := tfconv.Known(m.UnavailableConditions)
	archiveConditions := stringSliceFromList(m.ArchiveConditions)
	unavailableConditions := stringSliceFromList(m.UnavailableConditions)

	// Send both action sets or neither. The API replaces this object wholesale, so
	// omitting one key while setting the other would silently keep its old conditions
	// (and an empty list would never clear). Callers reach here via Adopt, which has
	// already hydrated any unset list from the live unit, so a total payload cannot
	// clobber a value the plan did not mention.
	if archiveKnown || unavailableKnown {
		out.InstallationReposConfig = &api_client.ShiftLeftInstallationReposConfig{
			ArchiveActions:     &api_client.ShiftLeftArchiveActions{Conditions: nonNilSlice(archiveConditions)},
			UnavailableActions: &api_client.ShiftLeftArchiveActions{Conditions: nonNilSlice(unavailableConditions)},
		}
	}

	return out
}

// nonNilSlice keeps an empty list serializing as [] rather than null.
func nonNilSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func FlattenConfigSettings(c api_client.ShiftLeftConfigSettings) ConfigSettingsModel {
	m := ConfigSettingsModel{
		PRSettingsModel: shift_left_common.PRSettingsModel{
			DisableScanPullRequests: types.BoolValue(c.DisableScanPullRequests),
			CommentsOnPullRequests:  tfconv.StringOrNull(c.CommentsOnPullRequests),
			PrSummaryComment:        tfconv.StringOrNull(c.PrSummaryComment),
			SkipCheckRuns:           tfconv.StringOrNull(c.SkipCheckRuns),
			ConfigFileSupport:       tfconv.StringOrNull(c.ConfigFileSupport),
		},
		PrSummaryAppendix:     tfconv.StringOrNull(c.PrSummaryAppendix),
		ArchiveConditions:     types.ListNull(types.StringType),
		UnavailableConditions: types.ListNull(types.StringType),
	}

	if c.InstallationReposConfig != nil {
		if c.InstallationReposConfig.ArchiveActions != nil {
			m.ArchiveConditions = stringSliceToList(c.InstallationReposConfig.ArchiveActions.Conditions)
		}
		if c.InstallationReposConfig.UnavailableActions != nil {
			m.UnavailableConditions = stringSliceToList(c.InstallationReposConfig.UnavailableActions.Conditions)
		}
	}

	return m
}

func MergeConfigSettings(base ConfigSettingsModel, overlay *ConfigSettingsModel) ConfigSettingsModel {
	if overlay == nil {
		return base
	}
	out := base
	if tfconv.Known(overlay.DisableScanPullRequests) {
		out.DisableScanPullRequests = overlay.DisableScanPullRequests
	}
	setStr := func(dst *types.String, src types.String) {
		if tfconv.Known(src) {
			*dst = src
		}
	}
	setStr(&out.CommentsOnPullRequests, overlay.CommentsOnPullRequests)
	setStr(&out.PrSummaryComment, overlay.PrSummaryComment)
	setStr(&out.SkipCheckRuns, overlay.SkipCheckRuns)
	setStr(&out.ConfigFileSupport, overlay.ConfigFileSupport)
	setStr(&out.PrSummaryAppendix, overlay.PrSummaryAppendix)
	if tfconv.Known(overlay.ArchiveConditions) {
		out.ArchiveConditions = overlay.ArchiveConditions
	}
	if tfconv.Known(overlay.UnavailableConditions) {
		out.UnavailableConditions = overlay.UnavailableConditions
	}
	return out
}
