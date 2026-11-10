package shift_left_common

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Embedded anonymously so tfsdk promotes the tags.
type PRSettingsModel struct {
	DisableScanPullRequests types.Bool   `tfsdk:"disable_scan_pull_requests"`
	CommentsOnPullRequests  types.String `tfsdk:"comments_on_pull_requests"`
	PrSummaryComment        types.String `tfsdk:"pr_summary_comment"`
	SkipCheckRuns           types.String `tfsdk:"skip_check_runs"`
	ConfigFileSupport       types.String `tfsdk:"config_file_support"`
}

var PRCommentValues = []string{"ALWAYS", "ONLY_ON_FAILED_ISSUES", "NEVER"}

// Full skip_check_runs enum; GitLab repositories use GitlabSkipCheckRunValues.
var FullSkipCheckRunValues = []string{"ALWAYS", "ONLY_ON_INTERNAL_ISSUE", "NEVER"}

var GitlabSkipCheckRunValues = []string{"ALWAYS", "NEVER"}

var ConfigFileSupportValues = []string{"ENABLED", "DISABLED"}

func PRCommentValidator() []validator.String {
	return []validator.String{stringvalidator.OneOf(PRCommentValues...)}
}

func SkipCheckRunsValidator(values []string) []validator.String {
	return []validator.String{stringvalidator.OneOf(values...)}
}

func ConfigFileSupportValidator() []validator.String {
	return []validator.String{stringvalidator.OneOf(ConfigFileSupportValues...)}
}
