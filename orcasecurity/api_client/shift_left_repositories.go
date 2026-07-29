package api_client

import "fmt"

// POST integrate returns empty body; re-list by SCM-side id.
// integrated_repositories has no DELETE — use repository_contexts.

type ScmRepository struct {
	ID             string
	ProjectID      string
	RepositoryName string
	RepositoryURL  string
	// Slug is Bitbucket-only; empty for the other providers.
	Slug                string
	Disabled            bool
	DisableScanPRs      *bool
	CommentsOnPRs       string
	PrSummaryComment    string
	SkipCheckRuns       string
	ConfigFileSupport   string
	Status              string
	RepositoryContextID string
	IntegrationStatus   string
	ScmPosturePolicyID  string
}

type scmRepoRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type scmIDRef struct {
	ID string `json:"id"`
}

type ScmRepositoryConfigUpdate struct {
	IDs                     []string `json:"ids"`
	Disabled                *bool    `json:"disabled,omitempty"`
	DisableScanPullRequests *bool    `json:"disable_scan_pull_requests,omitempty"`
	CommentsOnPullRequests  string   `json:"comments_on_pull_requests,omitempty"`
	PrSummaryComment        string   `json:"pr_summary_comment,omitempty"`
	SkipCheckRuns           string   `json:"skip_check_runs,omitempty"`
	ConfigFileSupport       string   `json:"config_file_support,omitempty"`
}

// ScmRepoIntegrationConfig is the batch-level configuration_settings sent with a
// repository integration POST so configuration is applied atomically at integrate
// time (rather than only via a follow-up PATCH, which would leave a window where
// the repository scans with default settings). `disabled` is intentionally not
// here: GitHub's integrate endpoint does not accept it, so it is applied
// post-integrate for all providers. The object is always sent (the API requires
// the field); its members are omitempty so unset values are left at API defaults.
type ScmRepoIntegrationConfig struct {
	DisableScanPullRequests *bool  `json:"disable_scan_pull_requests,omitempty"`
	CommentsOnPullRequests  string `json:"comments_on_pull_requests,omitempty"`
	PrSummaryComment        string `json:"pr_summary_comment,omitempty"`
	SkipCheckRuns           string `json:"skip_check_runs,omitempty"`
	ConfigFileSupport       string `json:"config_file_support,omitempty"`
}

func integratedRepositoriesPath(provider string) string {
	return fmt.Sprintf("/api/shiftleft/%s/integrated_repositories/", provider)
}

// SCM write helpers invalidate the list cache unconditionally: a PATCH/POST/DELETE
// that reached the server before erroring (e.g. a timeout) must not leave stale
// list data that the following find() would read back into state.
func (client *APIClient) updateScmRepositories(provider string, body ScmRepositoryConfigUpdate) error {
	_, err := client.Patch(integratedRepositoriesPath(provider), body)
	client.invalidateScmListCache()
	return err
}

func (client *APIClient) integrateScmRepositories(provider string, body any) error {
	_, err := client.Post(integratedRepositoriesPath(provider), body)
	client.invalidateScmListCache()
	return err
}

func (client *APIClient) DeleteRepositoryContext(repositoryContextID string) error {
	if repositoryContextID == "" {
		return fmt.Errorf("DeleteRepositoryContext: empty repository context id")
	}
	_, err := client.Delete(fmt.Sprintf("/api/shiftleft/repository_contexts/%s/", repositoryContextID))
	client.invalidateScmListCache()
	return err
}

func (client *APIClient) MoveRepositoryContexts(targetProjectID string, repositoryContextIDs []string) error {
	body := struct {
		TargetProjectID      string   `json:"target_project_id"`
		RepositoryContextIDs []string `json:"repository_context_ids"`
	}{targetProjectID, repositoryContextIDs}
	_, err := client.Post("/api/shiftleft/repository_contexts/move_project/", body)
	client.invalidateScmListCache()
	return err
}

type scmRepositoryDescriptor struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
}

// scmRepoIntegrateCommon holds the top-level fields every provider's repository
// integrate POST shares; providers embed it and add their key plus repositories.
type scmRepoIntegrateCommon struct {
	InstallationID        string                   `json:"installation_id"`
	ConfigurationSettings ScmRepoIntegrationConfig `json:"configuration_settings"`
	ProjectID             string                   `json:"project_id,omitempty"`
}

func newScmRepoIntegrateCommon(installationID, projectID string, cfg ScmRepoIntegrationConfig) scmRepoIntegrateCommon {
	return scmRepoIntegrateCommon{InstallationID: installationID, ConfigurationSettings: cfg, ProjectID: projectID}
}

type githubRepositoryItem struct {
	ID                      string     `json:"id"`
	GithubRepositoryID      int64      `json:"github_repository_id"`
	GithubInstallation      scmIDRef   `json:"github_installation"`
	Project                 *scmIDRef  `json:"project"`
	Repository              scmRepoRef `json:"repository"`
	Disabled                bool       `json:"disabled"`
	DisableScanPullRequests *bool      `json:"disable_scan_pull_requests"`
	CommentsOnPullRequests  string     `json:"comments_on_pull_requests"`
	PrSummaryComment        string     `json:"pr_summary_comment"`
	SkipCheckRuns           string     `json:"skip_check_runs"`
	ConfigFileSupport       string     `json:"config_file_support"`
	Status                  string     `json:"status"`
	RepositoryContextID     string     `json:"repository_context_id"`
	IntegrationStatus       string     `json:"integration_status"`
	ScmPosturePolicyID      string     `json:"scm_posture_policy_id"`
}

func (r *githubRepositoryItem) common() ScmRepository {
	return scmRepository(r.ID, r.Project, r.Repository, scmRepoConfig{
		Disabled:          r.Disabled,
		DisableScanPRs:    r.DisableScanPullRequests,
		CommentsOnPRs:     r.CommentsOnPullRequests,
		PrSummaryComment:  r.PrSummaryComment,
		SkipCheckRuns:     r.SkipCheckRuns,
		ConfigFileSupport: r.ConfigFileSupport,
	}, scmRepoStatus{r.Status, r.RepositoryContextID, r.IntegrationStatus, r.ScmPosturePolicyID})
}

func projectID(ref *scmIDRef) string {
	if ref == nil {
		return ""
	}
	return ref.ID
}

type scmRepoConfig struct {
	Disabled          bool
	DisableScanPRs    *bool
	CommentsOnPRs     string
	PrSummaryComment  string
	SkipCheckRuns     string
	ConfigFileSupport string
}

type scmRepoStatus struct {
	Status              string
	RepositoryContextID string
	IntegrationStatus   string
	ScmPosturePolicyID  string
}

func scmRepository(id string, project *scmIDRef, repo scmRepoRef, cfg scmRepoConfig, st scmRepoStatus) ScmRepository {
	return ScmRepository{
		ID:                  id,
		ProjectID:           projectID(project),
		RepositoryName:      repo.Name,
		RepositoryURL:       repo.URL,
		Disabled:            cfg.Disabled,
		DisableScanPRs:      cfg.DisableScanPRs,
		CommentsOnPRs:       cfg.CommentsOnPRs,
		PrSummaryComment:    cfg.PrSummaryComment,
		SkipCheckRuns:       cfg.SkipCheckRuns,
		ConfigFileSupport:   cfg.ConfigFileSupport,
		Status:              st.Status,
		RepositoryContextID: st.RepositoryContextID,
		IntegrationStatus:   st.IntegrationStatus,
		ScmPosturePolicyID:  st.ScmPosturePolicyID,
	}
}

type GithubRepositoryIntegrate struct {
	InstallationID     string
	GithubRepositoryID int64
	Name               string
	URL                string
	Branch             string
	ProjectID          string
	Config             ScmRepoIntegrationConfig
}

func (client *APIClient) IntegrateGithubRepository(req GithubRepositoryIntegrate) error {
	type repoEntry struct {
		scmRepositoryDescriptor
		GithubRepositoryID int64 `json:"github_repository_id"`
	}
	body := struct {
		scmRepoIntegrateCommon
		Repositories []repoEntry `json:"repositories"`
	}{
		newScmRepoIntegrateCommon(req.InstallationID, req.ProjectID, req.Config),
		[]repoEntry{{
			scmRepositoryDescriptor{Name: req.Name, URL: req.URL, Branch: req.Branch},
			req.GithubRepositoryID,
		}},
	}
	return client.integrateScmRepositories("github", body)
}

func (client *APIClient) FindGithubRepository(installationID string, githubRepositoryID int64) (*ScmRepository, error) {
	all, err := getAllScmPages[githubRepositoryItem](client, integratedRepositoriesPath("github"))
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].GithubInstallation.ID == installationID && all[i].GithubRepositoryID == githubRepositoryID {
			c := all[i].common()
			return &c, nil
		}
	}
	return nil, nil
}

func (client *APIClient) UpdateGithubRepositories(body ScmRepositoryConfigUpdate) error {
	return client.updateScmRepositories("github", body)
}

type gitlabRepositoryItem struct {
	ID                      string     `json:"id"`
	GitlabProjectID         int64      `json:"gitlab_project_id"`
	GitlabInstallation      scmIDRef   `json:"gitlab_installation"`
	Project                 *scmIDRef  `json:"project"`
	Repository              scmRepoRef `json:"repository"`
	Disabled                bool       `json:"disabled"`
	DisableScanPullRequests *bool      `json:"disable_scan_pull_requests"`
	CommentsOnPullRequests  string     `json:"comments_on_pull_requests"`
	PrSummaryComment        string     `json:"pr_summary_comment"`
	SkipCheckRuns           string     `json:"skip_check_runs"`
	ConfigFileSupport       string     `json:"config_file_support"`
	Status                  string     `json:"status"`
	RepositoryContextID     string     `json:"repository_context_id"`
	IntegrationStatus       string     `json:"integration_status"`
	ScmPosturePolicyID      string     `json:"scm_posture_policy_id"`
}

func (r *gitlabRepositoryItem) common() ScmRepository {
	return scmRepository(r.ID, r.Project, r.Repository, scmRepoConfig{
		Disabled:          r.Disabled,
		DisableScanPRs:    r.DisableScanPullRequests,
		CommentsOnPRs:     r.CommentsOnPullRequests,
		PrSummaryComment:  r.PrSummaryComment,
		SkipCheckRuns:     r.SkipCheckRuns,
		ConfigFileSupport: r.ConfigFileSupport,
	}, scmRepoStatus{r.Status, r.RepositoryContextID, r.IntegrationStatus, r.ScmPosturePolicyID})
}

type GitlabRepositoryIntegrate struct {
	InstallationID  string
	GitlabGroupID   int64
	GitlabProjectID int64
	Name            string
	URL             string
	Branch          string
	ProjectID       string
	Config          ScmRepoIntegrationConfig
}

func (client *APIClient) IntegrateGitlabRepository(req GitlabRepositoryIntegrate) error {
	type repoEntry struct {
		scmRepositoryDescriptor
		ID int64 `json:"id"`
	}
	body := struct {
		scmRepoIntegrateCommon
		GroupID      int64       `json:"group_id"`
		Repositories []repoEntry `json:"repositories"`
	}{
		newScmRepoIntegrateCommon(req.InstallationID, req.ProjectID, req.Config),
		req.GitlabGroupID,
		[]repoEntry{{
			scmRepositoryDescriptor{Name: req.Name, URL: req.URL, Branch: req.Branch},
			req.GitlabProjectID,
		}},
	}
	return client.integrateScmRepositories("gitlab", body)
}

func (client *APIClient) FindGitlabRepository(installationID string, gitlabProjectID int64) (*ScmRepository, error) {
	all, err := getAllScmPages[gitlabRepositoryItem](client, integratedRepositoriesPath("gitlab"))
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].GitlabInstallation.ID == installationID && all[i].GitlabProjectID == gitlabProjectID {
			c := all[i].common()
			return &c, nil
		}
	}
	return nil, nil
}

func (client *APIClient) UpdateGitlabRepositories(body ScmRepositoryConfigUpdate) error {
	return client.updateScmRepositories("gitlab", body)
}

type bitbucketRepositoryItem struct {
	ID                    string `json:"id"`
	BitbucketRepositoryID string `json:"bitbucket_repository_id"`
	BitbucketRepoSlug     string `json:"bitbucket_repository_slug"`
	AccountInstallation   struct {
		ID        string `json:"id"`
		AccountID string `json:"account_id"`
	} `json:"account_installation"`
	Project               *scmIDRef  `json:"project"`
	Repository            scmRepoRef `json:"repository"`
	Disabled              bool       `json:"disabled"`
	ConfigurationSettings struct {
		DisableScanPullRequests *bool  `json:"disable_scan_pull_requests"`
		CommentsOnPullRequests  string `json:"comments_on_pull_requests"`
		PrSummaryComment        string `json:"pr_summary_comment"`
		SkipCheckRuns           string `json:"skip_check_runs"`
		ConfigFileSupport       string `json:"config_file_support"`
	} `json:"configuration_settings"`
	Status              string `json:"status"`
	RepositoryContextID string `json:"repository_context_id"`
	IntegrationStatus   string `json:"integration_status"`
}

func (r *bitbucketRepositoryItem) common() ScmRepository {
	c := scmRepository(r.ID, r.Project, r.Repository, scmRepoConfig{
		Disabled:          r.Disabled,
		DisableScanPRs:    r.ConfigurationSettings.DisableScanPullRequests,
		CommentsOnPRs:     r.ConfigurationSettings.CommentsOnPullRequests,
		PrSummaryComment:  r.ConfigurationSettings.PrSummaryComment,
		SkipCheckRuns:     r.ConfigurationSettings.SkipCheckRuns,
		ConfigFileSupport: r.ConfigurationSettings.ConfigFileSupport,
	}, scmRepoStatus{r.Status, r.RepositoryContextID, r.IntegrationStatus, ""})
	c.Slug = r.BitbucketRepoSlug
	return c
}

type BitbucketRepositoryIntegrate struct {
	InstallationID        string
	AccountID             string
	BitbucketRepositoryID string
	Slug                  string
	Name                  string
	URL                   string
	Branch                string
	ProjectID             string
	Config                ScmRepoIntegrationConfig
}

func (client *APIClient) IntegrateBitbucketRepository(req BitbucketRepositoryIntegrate) error {
	type repoEntry struct {
		scmRepositoryDescriptor
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
	body := struct {
		scmRepoIntegrateCommon
		AccountID    string      `json:"account_id"`
		Repositories []repoEntry `json:"repositories"`
	}{
		newScmRepoIntegrateCommon(req.InstallationID, req.ProjectID, req.Config),
		req.AccountID,
		[]repoEntry{{
			scmRepositoryDescriptor{Name: req.Name, URL: req.URL, Branch: req.Branch},
			req.BitbucketRepositoryID,
			req.Slug,
		}},
	}
	return client.integrateScmRepositories("bitbucket", body)
}

// FindBitbucketRepository scopes the match to the parent installation's Orca
// account-installation id, not the external account slug: the same slug can be
// integrated under multiple installations in one org (slugs are unique only per
// installation, and bitbucket repository ids carry no uniqueness at all), so
// matching on the slug alone can return a repository under the wrong installation.
func (client *APIClient) FindBitbucketRepository(installationID, accountSlug, bitbucketRepositoryID string) (*ScmRepository, error) {
	account, err := client.FindBitbucketAccountBySlug(installationID, accountSlug)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, nil // account not integrated under this installation
	}
	all, err := getAllScmPages[bitbucketRepositoryItem](client, integratedRepositoriesPath("bitbucket"))
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].AccountInstallation.ID == account.ID && all[i].BitbucketRepositoryID == bitbucketRepositoryID {
			c := all[i].common()
			return &c, nil
		}
	}
	return nil, nil
}

func (client *APIClient) UpdateBitbucketRepositories(body ScmRepositoryConfigUpdate) error {
	return client.updateScmRepositories("bitbucket", body)
}

type azureRepositoryItem struct {
	ID                       string `json:"id"`
	AzureRepositoryID        string `json:"azure_repository_id"`
	AzureAccountInstallation struct {
		ID          string `json:"id"`
		AccountName string `json:"account_name"`
	} `json:"azure_account_installation"`
	Project               *scmIDRef  `json:"project"`
	Repository            scmRepoRef `json:"repository"`
	ManagedRepoProperties struct {
		Disabled          bool   `json:"disabled"`
		ConfigFileSupport string `json:"config_file_support"`
	} `json:"managed_repo_properties"`
	DisableScanPullRequests *bool  `json:"disable_scan_pull_requests"`
	CommentsOnPullRequests  string `json:"comments_on_pull_requests"`
	PrSummaryComment        string `json:"pr_summary_comment"`
	Status                  string `json:"status"`
	RepositoryContextID     string `json:"repository_context_id"`
	IntegrationStatus       string `json:"integration_status"`
	ScmPosturePolicyID      string `json:"scm_posture_policy_id"`
}

func (r *azureRepositoryItem) common() ScmRepository {
	return scmRepository(r.ID, r.Project, r.Repository, scmRepoConfig{
		Disabled:          r.ManagedRepoProperties.Disabled,
		DisableScanPRs:    r.DisableScanPullRequests,
		CommentsOnPRs:     r.CommentsOnPullRequests,
		PrSummaryComment:  r.PrSummaryComment,
		ConfigFileSupport: r.ManagedRepoProperties.ConfigFileSupport,
	}, scmRepoStatus{r.Status, r.RepositoryContextID, r.IntegrationStatus, r.ScmPosturePolicyID})
}

type AzureRepositoryIntegrate struct {
	InstallationID    string
	AccountName       string
	AzureRepositoryID string
	AzureProjectID    string
	Name              string
	URL               string
	Branch            string
	ProjectID         string
	Config            ScmRepoIntegrationConfig
}

func (client *APIClient) IntegrateAzureRepository(req AzureRepositoryIntegrate) error {
	type repoEntry struct {
		scmRepositoryDescriptor
		ID             string `json:"id"`
		AzureProjectID string `json:"azure_project_id"`
	}
	body := struct {
		scmRepoIntegrateCommon
		AzureAccountName string      `json:"azure_account_name"`
		Repositories     []repoEntry `json:"repositories"`
	}{
		newScmRepoIntegrateCommon(req.InstallationID, req.ProjectID, req.Config),
		req.AccountName,
		[]repoEntry{{
			scmRepositoryDescriptor{Name: req.Name, URL: req.URL, Branch: req.Branch},
			req.AzureRepositoryID,
			req.AzureProjectID,
		}},
	}
	return client.integrateScmRepositories("azure_devops", body)
}

// FindAzureRepository scopes the match to the parent installation's Orca
// account-installation id, not the external account name: an account name is
// unique only per installation and the same account can be integrated under
// multiple installations, so matching on the name alone can return a repository
// under the wrong installation.
func (client *APIClient) FindAzureRepository(installationID, accountName, azureRepositoryID string) (*ScmRepository, error) {
	account, err := client.FindAzureDevopsAccountByName(installationID, accountName)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, nil // account not integrated under this installation
	}
	all, err := getAllScmPages[azureRepositoryItem](client, integratedRepositoriesPath("azure_devops"))
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].AzureAccountInstallation.ID == account.ID && all[i].AzureRepositoryID == azureRepositoryID {
			c := all[i].common()
			return &c, nil
		}
	}
	return nil, nil
}

func (client *APIClient) UpdateAzureRepositories(body ScmRepositoryConfigUpdate) error {
	return client.updateScmRepositories("azure_devops", body)
}
