package api_client

type ShiftLeftArchiveActions struct {
	Conditions []string `json:"conditions,omitempty"`
}

type ShiftLeftInstallationReposConfig struct {
	ArchiveActions     *ShiftLeftArchiveActions `json:"archive_actions,omitempty"`
	UnavailableActions *ShiftLeftArchiveActions `json:"unavailable_actions,omitempty"`
}

type ShiftLeftConfigSettings struct {
	DisableScanPullRequests bool                              `json:"disable_scan_pull_requests"`
	CommentsOnPullRequests  string                            `json:"comments_on_pull_requests"`
	PrSummaryComment        string                            `json:"pr_summary_comment"`
	SkipCheckRuns           string                            `json:"skip_check_runs"`
	ConfigFileSupport       string                            `json:"config_file_support"`
	PrSummaryAppendix       string                            `json:"pr_summary_appendix"`
	InstallationReposConfig *ShiftLeftInstallationReposConfig `json:"installation_repositories_configuration,omitempty"`
}
