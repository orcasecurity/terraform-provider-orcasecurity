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
type ShiftLeftConfigSettings struct {
	DisableScanPullRequests bool                              `json:"disable_scan_pull_requests"`
	CommentsOnPullRequests  string                            `json:"comments_on_pull_requests,omitempty"`
	PrSummaryComment        string                            `json:"pr_summary_comment,omitempty"`
	SkipCheckRuns           string                            `json:"skip_check_runs,omitempty"`
	ConfigFileSupport       string                            `json:"config_file_support,omitempty"`
	PrSummaryAppendix       string                            `json:"pr_summary_appendix,omitempty"`
	InstallationReposConfig *ShiftLeftInstallationReposConfig `json:"installation_repositories_configuration,omitempty"`
}
