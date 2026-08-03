package shift_left_common

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// PR-behavior enums and validators shared by unit-level (account/group) and
// repository-level configuration: the same wire concepts appear on both surfaces.

// PRCommentValues is the enum shared by comments_on_pull_requests and pr_summary_comment.
var PRCommentValues = []string{"ALWAYS", "ONLY_ON_FAILED_ISSUES", "NEVER"}

// FullSkipCheckRunValues is the three-valued skip_check_runs enum used everywhere
// except GitLab repositories, which only support ALWAYS/NEVER.
var FullSkipCheckRunValues = []string{"ALWAYS", "ONLY_ON_INTERNAL_ISSUE", "NEVER"}

// GitlabSkipCheckRunValues is GitLab's repository-level skip_check_runs restriction.
var GitlabSkipCheckRunValues = []string{"ALWAYS", "NEVER"}

// ConfigFileSupportValues is the shared config_file_support enum.
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
