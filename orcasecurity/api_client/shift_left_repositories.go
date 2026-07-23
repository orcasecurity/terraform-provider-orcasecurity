package api_client

import "fmt"

// This file covers the per-repository integration lifecycle shared by all SCM
// providers:
//
//   - integrate:  POST /api/shiftleft/{provider}/integrated_repositories/
//     The response body is empty, so callers re-list and match by the
//     SCM-side repository id. Unit-level fields on the POST body
//     (installation_mode, project_id, policies, configuration_settings) only
//     apply when the owning unit is being created by this call; integrating
//     into an existing unit leaves the unit untouched.
//   - update:     PATCH .../integrated_repositories/ with {ids: [...]} bulk
//     config body. Null fields mean "leave unchanged".
//   - remove:     DELETE /api/shiftleft/repository_contexts/{ctx}/ — the
//     integrated_repositories endpoints define no DELETE route, and PATCHing
//     disabled=true merely pauses scanning.
//   - move:       POST /api/shiftleft/repository_contexts/move_project/.

// ScmRepository is the provider-neutral view of one integrated repository;
// each provider DTO normalizes into it.
type ScmRepository struct {
	ID                  string
	UnitID              string // owning unit row (installation/group/account)
	ProjectID           string
	RepositoryName      string
	RepositoryURL       string
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

// ScmRepositoryConfigUpdate is the shared bulk PATCH body. Nil pointers are
// omitted, which the API treats as "leave unchanged" (an explicit null can
// therefore not clear an override).
type ScmRepositoryConfigUpdate struct {
	IDs                     []string `json:"ids"`
	Disabled                *bool    `json:"disabled,omitempty"`
	DisableScanPullRequests *bool    `json:"disable_scan_pull_requests,omitempty"`
	CommentsOnPullRequests  string   `json:"comments_on_pull_requests,omitempty"`
	PrSummaryComment        string   `json:"pr_summary_comment,omitempty"`
	SkipCheckRuns           string   `json:"skip_check_runs,omitempty"`
	ConfigFileSupport       string   `json:"config_file_support,omitempty"`
}

func integratedRepositoriesPath(provider string) string {
	return fmt.Sprintf("/api/shiftleft/%s/integrated_repositories/", provider)
}

func (client *APIClient) updateScmRepositories(provider string, body ScmRepositoryConfigUpdate) error {
	_, err := client.Patch(integratedRepositoriesPath(provider), body)
	if err == nil {
		client.invalidateScmListCache()
	}
	return err
}

func (client *APIClient) integrateScmRepositories(provider string, body any) error {
	_, err := client.Post(integratedRepositoriesPath(provider), body)
	if err == nil {
		client.invalidateScmListCache()
	}
	return err
}

// DeleteRepositoryContext un-integrates a repository entirely.
func (client *APIClient) DeleteRepositoryContext(repositoryContextID string) error {
	_, err := client.Delete(fmt.Sprintf("/api/shiftleft/repository_contexts/%s/", repositoryContextID))
	if err == nil {
		client.invalidateScmListCache()
	}
	return err
}

// MoveRepositoryContexts re-assigns repositories to another Shift Left
// project.
func (client *APIClient) MoveRepositoryContexts(targetProjectID string, repositoryContextIDs []string) error {
	body := struct {
		TargetProjectID      string   `json:"target_project_id"`
		RepositoryContextIDs []string `json:"repository_context_ids"`
	}{targetProjectID, repositoryContextIDs}
	_, err := client.Post("/api/shiftleft/repository_contexts/move_project/", body)
	if err == nil {
		client.invalidateScmListCache()
	}
	return err
}

// scmRepositoryDescriptor is the shared shape of one repository entry in an
// integrate POST. Provider-specific ids ride alongside in wrapper structs.
type scmRepositoryDescriptor struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
}

// --- GitHub ---

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
	return ScmRepository{
		ID:                  r.ID,
		UnitID:              r.GithubInstallation.ID,
		ProjectID:           projectID(r.Project),
		RepositoryName:      r.Repository.Name,
		RepositoryURL:       r.Repository.URL,
		Disabled:            r.Disabled,
		DisableScanPRs:      r.DisableScanPullRequests,
		CommentsOnPRs:       r.CommentsOnPullRequests,
		PrSummaryComment:    r.PrSummaryComment,
		SkipCheckRuns:       r.SkipCheckRuns,
		ConfigFileSupport:   r.ConfigFileSupport,
		Status:              r.Status,
		RepositoryContextID: r.RepositoryContextID,
		IntegrationStatus:   r.IntegrationStatus,
		ScmPosturePolicyID:  r.ScmPosturePolicyID,
	}
}

func projectID(ref *scmIDRef) string {
	if ref == nil {
		return ""
	}
	return ref.ID
}

// GithubRepositoryIntegrate is the create request for one GitHub repository.
type GithubRepositoryIntegrate struct {
	InstallationID     string
	GithubRepositoryID int64
	Name               string
	URL                string
	Branch             string
	ProjectID          string
}

func (client *APIClient) IntegrateGithubRepository(req GithubRepositoryIntegrate) error {
	type repoEntry struct {
		scmRepositoryDescriptor
		GithubRepositoryID int64 `json:"github_repository_id"`
	}
	body := struct {
		InstallationID        string      `json:"installation_id"`
		ConfigurationSettings struct{}    `json:"configuration_settings"`
		ProjectID             string      `json:"project_id,omitempty"`
		Repositories          []repoEntry `json:"repositories"`
	}{
		InstallationID: req.InstallationID,
		ProjectID:      req.ProjectID,
		Repositories: []repoEntry{{
			scmRepositoryDescriptor{Name: req.Name, URL: req.URL, Branch: req.Branch},
			req.GithubRepositoryID,
		}},
	}
	return client.integrateScmRepositories("github", body)
}

// FindGithubRepository locates an integrated repository by its GitHub-side id
// within one installation. Returns nil when absent.
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

// --- GitLab ---

type gitlabRepositoryItem struct {
	ID                      string     `json:"id"`
	GitlabProjectID         int64      `json:"gitlab_project_id"`
	GitlabInstallation      scmIDRef   `json:"gitlab_installation"`
	GitlabGroup             scmIDRef   `json:"gitlab_group"`
	GroupInstallationID     string     `json:"gitlab_group_installation_id"`
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
	return ScmRepository{
		ID:                  r.ID,
		UnitID:              r.GroupInstallationID,
		ProjectID:           projectID(r.Project),
		RepositoryName:      r.Repository.Name,
		RepositoryURL:       r.Repository.URL,
		Disabled:            r.Disabled,
		DisableScanPRs:      r.DisableScanPullRequests,
		CommentsOnPRs:       r.CommentsOnPullRequests,
		PrSummaryComment:    r.PrSummaryComment,
		SkipCheckRuns:       r.SkipCheckRuns,
		ConfigFileSupport:   r.ConfigFileSupport,
		Status:              r.Status,
		RepositoryContextID: r.RepositoryContextID,
		IntegrationStatus:   r.IntegrationStatus,
		ScmPosturePolicyID:  r.ScmPosturePolicyID,
	}
}

// GitlabRepositoryIntegrate is the create request for one GitLab project
// (repository). GitlabGroupID/GitlabProjectID are GitLab-side numeric ids.
type GitlabRepositoryIntegrate struct {
	InstallationID  string
	GitlabGroupID   int64
	GitlabProjectID int64
	Name            string
	URL             string
	Branch          string
	ProjectID       string
}

func (client *APIClient) IntegrateGitlabRepository(req GitlabRepositoryIntegrate) error {
	type repoEntry struct {
		scmRepositoryDescriptor
		ID int64 `json:"id"`
	}
	body := struct {
		InstallationID        string      `json:"installation_id"`
		GroupID               int64       `json:"group_id"`
		ConfigurationSettings struct{}    `json:"configuration_settings"`
		ProjectID             string      `json:"project_id,omitempty"`
		Repositories          []repoEntry `json:"repositories"`
	}{
		InstallationID: req.InstallationID,
		GroupID:        req.GitlabGroupID,
		ProjectID:      req.ProjectID,
		Repositories: []repoEntry{{
			scmRepositoryDescriptor{Name: req.Name, URL: req.URL, Branch: req.Branch},
			req.GitlabProjectID,
		}},
	}
	return client.integrateScmRepositories("gitlab", body)
}

// FindGitlabRepository locates an integrated GitLab project by its
// GitLab-side numeric id within one installation. Returns nil when absent.
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

// --- Bitbucket ---

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
	return ScmRepository{
		ID:                  r.ID,
		UnitID:              r.AccountInstallation.ID,
		ProjectID:           projectID(r.Project),
		RepositoryName:      r.Repository.Name,
		RepositoryURL:       r.Repository.URL,
		Disabled:            r.Disabled,
		DisableScanPRs:      r.ConfigurationSettings.DisableScanPullRequests,
		CommentsOnPRs:       r.ConfigurationSettings.CommentsOnPullRequests,
		PrSummaryComment:    r.ConfigurationSettings.PrSummaryComment,
		SkipCheckRuns:       r.ConfigurationSettings.SkipCheckRuns,
		ConfigFileSupport:   r.ConfigurationSettings.ConfigFileSupport,
		Status:              r.Status,
		RepositoryContextID: r.RepositoryContextID,
		IntegrationStatus:   r.IntegrationStatus,
	}
}

// BitbucketRepositoryIntegrate is the create request for one Bitbucket
// repository. AccountID is the workspace (cloud) or project (server) slug;
// BitbucketRepositoryID/Slug are Bitbucket-side identifiers.
type BitbucketRepositoryIntegrate struct {
	InstallationID        string
	AccountID             string
	BitbucketRepositoryID string
	Slug                  string
	Name                  string
	URL                   string
	Branch                string
	ProjectID             string
}

func (client *APIClient) IntegrateBitbucketRepository(req BitbucketRepositoryIntegrate) error {
	type repoEntry struct {
		scmRepositoryDescriptor
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
	body := struct {
		InstallationID        string      `json:"installation_id"`
		AccountID             string      `json:"account_id"`
		ConfigurationSettings struct{}    `json:"configuration_settings"`
		ProjectID             string      `json:"project_id,omitempty"`
		Repositories          []repoEntry `json:"repositories"`
	}{
		InstallationID: req.InstallationID,
		AccountID:      req.AccountID,
		ProjectID:      req.ProjectID,
		Repositories: []repoEntry{{
			scmRepositoryDescriptor{Name: req.Name, URL: req.URL, Branch: req.Branch},
			req.BitbucketRepositoryID,
			req.Slug,
		}},
	}
	return client.integrateScmRepositories("bitbucket", body)
}

// FindBitbucketRepository locates an integrated repository by its
// Bitbucket-side id within one account (workspace/project slug). Returns nil
// when absent.
func (client *APIClient) FindBitbucketRepository(accountID, bitbucketRepositoryID string) (*ScmRepository, error) {
	all, err := getAllScmPages[bitbucketRepositoryItem](client, integratedRepositoriesPath("bitbucket"))
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].AccountInstallation.AccountID == accountID && all[i].BitbucketRepositoryID == bitbucketRepositoryID {
			c := all[i].common()
			return &c, nil
		}
	}
	return nil, nil
}

func (client *APIClient) UpdateBitbucketRepositories(body ScmRepositoryConfigUpdate) error {
	return client.updateScmRepositories("bitbucket", body)
}

// --- Azure DevOps ---

type azureRepositoryItem struct {
	ID                       string `json:"id"`
	AzureRepositoryID        string `json:"azure_repository_id"`
	AzureAccountInstallation struct {
		ID          string `json:"id"`
		AccountName string `json:"account_name"`
	} `json:"azure_account_installation"`
	Project              *scmIDRef  `json:"project"`
	Repository           scmRepoRef `json:"repository"`
	ManagedRepoProperies struct {
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
	return ScmRepository{
		ID:                  r.ID,
		UnitID:              r.AzureAccountInstallation.ID,
		ProjectID:           projectID(r.Project),
		RepositoryName:      r.Repository.Name,
		RepositoryURL:       r.Repository.URL,
		Disabled:            r.ManagedRepoProperies.Disabled,
		DisableScanPRs:      r.DisableScanPullRequests,
		CommentsOnPRs:       r.CommentsOnPullRequests,
		PrSummaryComment:    r.PrSummaryComment,
		ConfigFileSupport:   r.ManagedRepoProperies.ConfigFileSupport,
		Status:              r.Status,
		RepositoryContextID: r.RepositoryContextID,
		IntegrationStatus:   r.IntegrationStatus,
		ScmPosturePolicyID:  r.ScmPosturePolicyID,
	}
}

// AzureRepositoryIntegrate is the create request for one Azure DevOps
// repository. AzureRepositoryID/AzureProjectID are Azure-side UUIDs;
// AccountName is the Azure DevOps organization name.
type AzureRepositoryIntegrate struct {
	InstallationID    string
	AccountName       string
	AzureRepositoryID string
	AzureProjectID    string
	Name              string
	URL               string
	Branch            string
	ProjectID         string
}

func (client *APIClient) IntegrateAzureRepository(req AzureRepositoryIntegrate) error {
	type repoEntry struct {
		scmRepositoryDescriptor
		ID             string `json:"id"`
		AzureProjectID string `json:"azure_project_id"`
	}
	body := struct {
		InstallationID        string      `json:"installation_id"`
		AzureAccountName      string      `json:"azure_account_name"`
		ConfigurationSettings struct{}    `json:"configuration_settings"`
		ProjectID             string      `json:"project_id,omitempty"`
		Repositories          []repoEntry `json:"repositories"`
	}{
		InstallationID:   req.InstallationID,
		AzureAccountName: req.AccountName,
		ProjectID:        req.ProjectID,
		Repositories: []repoEntry{{
			scmRepositoryDescriptor{Name: req.Name, URL: req.URL, Branch: req.Branch},
			req.AzureRepositoryID,
			req.AzureProjectID,
		}},
	}
	return client.integrateScmRepositories("azure_devops", body)
}

// FindAzureRepository locates an integrated repository by its Azure-side UUID
// within one account (organization). Returns nil when absent.
func (client *APIClient) FindAzureRepository(accountName, azureRepositoryID string) (*ScmRepository, error) {
	all, err := getAllScmPages[azureRepositoryItem](client, integratedRepositoriesPath("azure_devops"))
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].AzureAccountInstallation.AccountName == accountName && all[i].AzureRepositoryID == azureRepositoryID {
			c := all[i].common()
			return &c, nil
		}
	}
	return nil, nil
}

func (client *APIClient) UpdateAzureRepositories(body ScmRepositoryConfigUpdate) error {
	return client.updateScmRepositories("azure_devops", body)
}
