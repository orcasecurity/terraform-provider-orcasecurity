// Package shift_left_integration holds the shared `configuration_settings`
// schema, model, and expand/flatten helpers reused by the per-SCM-provider
// shift-left integration resources (GitHub, GitLab, Azure DevOps, Bitbucket).
package shift_left_integration

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConfigSettingsModel is the Terraform representation of the
// configuration_settings object sent/received by the Orca SCM integration
// API. It is embedded as a nested attribute on each per-provider resource
// schema (github/gitlab/azure/bitbucket), gated via FieldGate for fields that
// are not supported by every provider.
type ConfigSettingsModel struct {
	DisableScanPullRequests types.Bool     `tfsdk:"disable_scan_pull_requests"`
	CommentsOnPullRequests  types.String   `tfsdk:"comments_on_pull_requests"`
	PrSummaryComment        types.String   `tfsdk:"pr_summary_comment"`
	SkipCheckRuns           types.String   `tfsdk:"skip_check_runs"`
	ConfigFileSupport       types.String   `tfsdk:"config_file_support"`
	PrSummaryAppendix       types.String   `tfsdk:"pr_summary_appendix"`
	ArchiveConditions       []types.String `tfsdk:"archive_conditions"`
	UnavailableConditions   []types.String `tfsdk:"unavailable_conditions"`
}

// FieldGate controls which configuration_settings fields are exposed by a
// given per-provider resource schema, since not every SCM integration
// supports every field:
//   - ArchiveActions: gates both archive_conditions and unavailable_conditions
//     (installation_repositories_configuration), which are only meaningful for
//     providers that model installation/repository access changes.
type FieldGate struct {
	ArchiveActions bool
}

// ConfigSettingsAttributes returns the nested attribute map for the
// configuration_settings object, omitting fields not enabled by opts. All
// attributes are Optional+Computed: the server always returns a value for
// every field, and per-provider resources adopt existing units, so the
// server is authoritative and null-vs-config plan drift must be avoided.
func ConfigSettingsAttributes(opts FieldGate) map[string]schema.Attribute {
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
				stringvalidator.OneOf("ALWAYS", "NEVER", "ONLY_ON_FAILED_SCAN"),
			},
		},
		"skip_check_runs": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "When to skip posting check runs.",
			Validators: []validator.String{
				stringvalidator.OneOf("ALWAYS", "NEVER", "ONLY_ON_INTERNAL_ISSUE"),
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

	if opts.ArchiveActions {
		attrs["archive_conditions"] = schema.ListAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: "Conditions that trigger an archive action for repositories (installation_repositories_configuration.archive_actions.conditions).",
			Validators: []validator.List{
				listvalidator.ValueStringsAre(stringvalidator.OneOf("AVOID_SCAN", "DELETE_REPO")),
			},
		}
		attrs["unavailable_conditions"] = schema.ListAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: "Conditions that trigger an action when a repository becomes unavailable (installation_repositories_configuration.unavailable_actions.conditions).",
			Validators: []validator.List{
				listvalidator.ValueStringsAre(stringvalidator.OneOf("DELETE_REPO")),
			},
		}
	}

	return attrs
}

// stringSliceFromTypes converts a []types.String into a []string, dropping
// null/unknown entries.
func stringSliceFromTypes(values []types.String) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, v := range values {
		if !v.IsNull() && !v.IsUnknown() {
			result = append(result, v.ValueString())
		}
	}
	return result
}

// stringSliceToTypes converts a []string into a []types.String. Returns nil
// for an empty input so the resulting model attribute stays null rather than
// an empty list.
func stringSliceToTypes(values []string) []types.String {
	if len(values) == 0 {
		return nil
	}
	result := make([]types.String, 0, len(values))
	for _, v := range values {
		result = append(result, types.StringValue(v))
	}
	return result
}

// ExpandConfigSettings converts a ConfigSettingsModel into the API payload
// shape. InstallationReposConfig (and its archive_actions/unavailable_actions
// children) is only built when the corresponding conditions list is
// non-empty, so unused providers don't send an empty nested object.
func ExpandConfigSettings(m *ConfigSettingsModel) api_client.ShiftLeftConfigSettings {
	if m == nil {
		return api_client.ShiftLeftConfigSettings{}
	}

	out := api_client.ShiftLeftConfigSettings{
		DisableScanPullRequests: m.DisableScanPullRequests.ValueBool(),
		CommentsOnPullRequests:  m.CommentsOnPullRequests.ValueString(),
		PrSummaryComment:        m.PrSummaryComment.ValueString(),
		SkipCheckRuns:           m.SkipCheckRuns.ValueString(),
		ConfigFileSupport:       m.ConfigFileSupport.ValueString(),
		PrSummaryAppendix:       m.PrSummaryAppendix.ValueString(),
	}

	archiveConditions := stringSliceFromTypes(m.ArchiveConditions)
	unavailableConditions := stringSliceFromTypes(m.UnavailableConditions)

	if len(archiveConditions) > 0 || len(unavailableConditions) > 0 {
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
	}

	return out
}

// FlattenConfigSettings converts the API payload shape back into a
// ConfigSettingsModel, reversing ExpandConfigSettings.
func FlattenConfigSettings(c api_client.ShiftLeftConfigSettings) ConfigSettingsModel {
	m := ConfigSettingsModel{
		DisableScanPullRequests: types.BoolValue(c.DisableScanPullRequests),
		CommentsOnPullRequests:  optionalString(c.CommentsOnPullRequests),
		PrSummaryComment:        optionalString(c.PrSummaryComment),
		SkipCheckRuns:           optionalString(c.SkipCheckRuns),
		ConfigFileSupport:       optionalString(c.ConfigFileSupport),
		PrSummaryAppendix:       optionalString(c.PrSummaryAppendix),
	}

	if c.InstallationReposConfig != nil {
		if c.InstallationReposConfig.ArchiveActions != nil {
			m.ArchiveConditions = stringSliceToTypes(c.InstallationReposConfig.ArchiveActions.Conditions)
		}
		if c.InstallationReposConfig.UnavailableActions != nil {
			m.UnavailableConditions = stringSliceToTypes(c.InstallationReposConfig.UnavailableActions.Conditions)
		}
	}

	return m
}

// optionalString maps an API string into a nullable state value: an empty
// string (attribute not set / not returned) becomes null so it matches an
// unset Optional attribute and does not produce a spurious plan diff.
func optionalString(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}
