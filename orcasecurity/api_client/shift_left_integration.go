package api_client

// Conditions has no omitempty: an empty list is how a condition set is cleared, and
// omitting the key leaves the previous conditions in place server-side.
type ShiftLeftArchiveActions struct {
	Conditions []string `json:"conditions"`
}

// The API replaces installation_repositories_configuration wholesale, so writers must
// populate both action sets whenever the parent object is sent.
type ShiftLeftInstallationReposConfig struct {
	ArchiveActions     *ShiftLeftArchiveActions `json:"archive_actions,omitempty"`
	UnavailableActions *ShiftLeftArchiveActions `json:"unavailable_actions,omitempty"`
}

type scmUnitIntegrateBody struct {
	InstallationID        string                  `json:"installation_id"`
	InstallationMode      string                  `json:"installation_mode,omitempty"`
	DefaultPolicies       bool                    `json:"default_policies"`
	Policies              []string                `json:"policies"`
	ProjectID             string                  `json:"project_id,omitempty"`
	ConfigurationSettings ShiftLeftConfigSettings `json:"configuration_settings"`
}

func newScmUnitIntegrateBody(installationID string, b ScmInstallationUpdate) scmUnitIntegrateBody {
	return scmUnitIntegrateBody{
		InstallationID:        installationID,
		InstallationMode:      b.InstallationMode,
		DefaultPolicies:       b.DefaultPolicies,
		Policies:              b.Policies,
		ProjectID:             b.ProjectID,
		ConfigurationSettings: b.ConfigSettings,
	}
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
