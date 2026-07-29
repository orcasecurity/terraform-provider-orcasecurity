package shift_left_integration

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConfigSettingsModel struct {
	DisableScanPullRequests types.Bool   `tfsdk:"disable_scan_pull_requests"`
	CommentsOnPullRequests  types.String `tfsdk:"comments_on_pull_requests"`
	PrSummaryComment        types.String `tfsdk:"pr_summary_comment"`
	SkipCheckRuns           types.String `tfsdk:"skip_check_runs"`
	ConfigFileSupport       types.String `tfsdk:"config_file_support"`
	PrSummaryAppendix       types.String `tfsdk:"pr_summary_appendix"`
	ArchiveConditions       types.List   `tfsdk:"archive_conditions"`
	UnavailableConditions   types.List   `tfsdk:"unavailable_conditions"`
}

// accountSkipCheckRuns is the same on every provider: the account/group PUT
// types skip_check_runs as PerformActionStatus, which is three-valued. The
// two-value ALWAYS/NEVER enum (GitlabPerformActionStatus) is the GitLab
// *repository*-level contract only, and lives in shift_left_repository.
var accountSkipCheckRuns = []string{"ALWAYS", "NEVER", "ONLY_ON_INTERNAL_ISSUE"}

func ConfigSettingsAttributes() map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"disable_scan_pull_requests": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Disable scanning pull requests.",
		},
		"comments_on_pull_requests": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "When to post scan result comments on pull requests.",
			Validators: []validator.String{
				stringvalidator.OneOf("ALWAYS", "NEVER", "ONLY_ON_FAILED_ISSUES"),
			},
		},
		"pr_summary_comment": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "When to post a pull request summary comment.",
			Validators: []validator.String{
				stringvalidator.OneOf("ALWAYS", "ONLY_ON_FAILED_ISSUES", "NEVER"),
			},
		},
		"skip_check_runs": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "When to skip posting check runs.",
			Validators: []validator.String{
				stringvalidator.OneOf(accountSkipCheckRuns...),
			},
		},
		"config_file_support": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether in-repo Orca config file support is enabled.",
			Validators: []validator.String{
				stringvalidator.OneOf("ENABLED", "DISABLED"),
			},
		},
		"pr_summary_appendix": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Additional free-text appendix appended to the pull request summary comment.",
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
	}
	attrs["unavailable_conditions"] = schema.ListAttribute{
		Optional:    true,
		Computed:    true,
		ElementType: types.StringType,
		Description: "Conditions that trigger an action when a repository becomes unavailable (installation_repositories_configuration.unavailable_actions.conditions). API accepts AVOID_SCAN and DELETE_REPO (same as archive_conditions).",
		Validators: []validator.List{
			listvalidator.ValueStringsAre(stringvalidator.OneOf("AVOID_SCAN", "DELETE_REPO")),
		},
	}

	return attrs
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

// ExpandConfigSettings builds the configuration_settings body for the account
// PUT.
//
// The four enum fields are required server-side and rejected when empty, yet the
// API returns "" for them on some legacy units — which FlattenConfigSettings maps
// to null and this would otherwise send straight back as "", failing the whole
// update with a 400. Unset enums therefore fall back to the same defaults a
// create uses, mirroring normalizeInstallationMode for the mode field.
//
// pr_summary_appendix is deliberately not defaulted: it is optional server-side
// and "" is meaningful there ("clear the appendix").
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

	switch {
	case len(archiveConditions) > 0 || len(unavailableConditions) > 0:
		installationReposConfig := &api_client.ShiftLeftInstallationReposConfig{}
		if len(archiveConditions) > 0 {
			installationReposConfig.ArchiveActions = &api_client.ShiftLeftArchiveActions{
				Conditions: archiveConditions,
			}
		}
		if len(unavailableConditions) > 0 {
			installationReposConfig.UnavailableActions = &api_client.ShiftLeftArchiveActions{
				Conditions: unavailableConditions,
			}
		}
		out.InstallationReposConfig = installationReposConfig
	case archiveKnown || unavailableKnown:
		// Explicit empty lists (UI toggles off) must clear server-side config.
		// Send {} rather than omitting the key — backend update paths assign
		// model_dump() values including this field.
		out.InstallationReposConfig = &api_client.ShiftLeftInstallationReposConfig{}
	}

	return out
}

func FlattenConfigSettings(c api_client.ShiftLeftConfigSettings) ConfigSettingsModel {
	m := ConfigSettingsModel{
		DisableScanPullRequests: types.BoolValue(c.DisableScanPullRequests),
		CommentsOnPullRequests:  tfconv.StringOrNull(c.CommentsOnPullRequests),
		PrSummaryComment:        tfconv.StringOrNull(c.PrSummaryComment),
		SkipCheckRuns:           tfconv.StringOrNull(c.SkipCheckRuns),
		ConfigFileSupport:       tfconv.StringOrNull(c.ConfigFileSupport),
		PrSummaryAppendix:       tfconv.StringOrNull(c.PrSummaryAppendix),
		ArchiveConditions:       types.ListNull(types.StringType),
		UnavailableConditions:   types.ListNull(types.StringType),
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
