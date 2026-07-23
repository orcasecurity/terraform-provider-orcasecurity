package api_client

// ShiftLeftArchiveActions represents the "conditions" list nested under either
// archive_actions or unavailable_actions in installation_repositories_configuration.
type ShiftLeftArchiveActions struct {
	Conditions []string `json:"conditions,omitempty"`
}

// ShiftLeftInstallationReposConfig maps to installation_repositories_configuration
// in the SCM integration configuration_settings payload.
type ShiftLeftInstallationReposConfig struct {
	ArchiveActions     *ShiftLeftArchiveActions `json:"archive_actions,omitempty"`
	UnavailableActions *ShiftLeftArchiveActions `json:"unavailable_actions,omitempty"`
}

// ShiftLeftConfigSettings mirrors the configuration_settings object sent/received
// by the Orca SCM integration UI/API for shift-left source-control integrations
// (GitHub, GitLab, Azure DevOps, Bitbucket).
//
// The API's update contract (UpdateConfigurationSettingsRequest) requires
// disable_scan_pull_requests, comments_on_pull_requests, pr_summary_comment,
// skip_check_runs, and config_file_support on every PUT, so none of those
// carry omitempty: the adopt write path guarantees they hold live values, and
// the JSON contract should not silently allow dropping a required field.
// pr_summary_appendix is likewise always sent (even when empty) to match the
// UI, which writes `pr_summary_appendix: customPrNote ?? ”` on every update.
// Only installation_repositories_configuration is optional server-side.
type ShiftLeftConfigSettings struct {
	DisableScanPullRequests bool                              `json:"disable_scan_pull_requests"`
	CommentsOnPullRequests  string                            `json:"comments_on_pull_requests"`
	PrSummaryComment        string                            `json:"pr_summary_comment"`
	SkipCheckRuns           string                            `json:"skip_check_runs"`
	ConfigFileSupport       string                            `json:"config_file_support"`
	PrSummaryAppendix       string                            `json:"pr_summary_appendix"`
	InstallationReposConfig *ShiftLeftInstallationReposConfig `json:"installation_repositories_configuration,omitempty"`
}
