package api_client

import (
	"fmt"
	"strconv"
)

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

// Sent on integrate POST (atomic config). disabled omitted — GitHub integrate rejects it; applied via follow-up PATCH.
type ScmRepoIntegrationConfig struct {
	DisableScanPullRequests *bool  `json:"disable_scan_pull_requests,omitempty"`
	CommentsOnPullRequests  string `json:"comments_on_pull_requests,omitempty"`
	PrSummaryComment        string `json:"pr_summary_comment,omitempty"`
	SkipCheckRuns           string `json:"skip_check_runs,omitempty"`
	ConfigFileSupport       string `json:"config_file_support,omitempty"`
}

// PATCH body: ids + disabled + config. Anonymous embed keeps a flat JSON object.
type ScmRepositoryConfigUpdate struct {
	IDs      []string `json:"ids"`
	Disabled *bool    `json:"disabled,omitempty"`
	ScmRepoIntegrationConfig
}

func integratedRepositoriesPath(provider string) string {
	return fmt.Sprintf("/api/shiftleft/%s/integrated_repositories/", provider)
}

// List with optional server filters, then match locally — unknown filter keys are silently ignored.
func findScmRepository[T any](
	client *APIClient,
	provider string,
	filters listFilters,
	match func(*T) bool,
	common func(*T) ScmRepository,
) (*ScmRepository, error) {
	all, err := getAllScmPages[T](client, integratedRepositoriesPath(provider), filters)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if match(&all[i]) {
			repo := common(&all[i])
			return &repo, nil
		}
	}
	return nil, nil
}

// Bitbucket and Azure repositories are addressed through their owning Orca account-installation row,
// so a missing account means the repository's existence is unknown, not that it was removed. Reporting
// absence would make Terraform drop a live integration from state and then try to re-integrate an
// already-integrated repository, so the ambiguity has to surface as an error the operator can act on.
func errAccountLookupFailed(scmName, accountKind, account, installationID string) error {
	return fmt.Errorf(
		"%s %s %q is not integrated under installation %s, so its repositories cannot be looked up. "+
			"Check installation_id and account_id; if the account was de-integrated its repositories are "+
			"gone too, so drop the repository from state with `terraform state rm`",
		scmName, accountKind, account, installationID)
}

func (client *APIClient) updateScmRepositories(provider string, body ScmRepositoryConfigUpdate) error {
	_, err := client.Patch(integratedRepositoriesPath(provider), body)
	return err
}

func (client *APIClient) integrateScmRepositories(provider string, body any) error {
	_, err := client.Post(integratedRepositoriesPath(provider), body)
	return err
}

func (client *APIClient) DeleteRepositoryContext(repositoryContextID string) error {
	if repositoryContextID == "" {
		return fmt.Errorf("DeleteRepositoryContext: empty repository context id")
	}
	_, err := client.Delete(fmt.Sprintf("/api/shiftleft/repository_contexts/%s/", repositoryContextID))
	return err
}

func (client *APIClient) MoveRepositoryContexts(targetProjectID string, repositoryContextIDs []string) error {
	body := struct {
		TargetProjectID      string   `json:"target_project_id"`
		RepositoryContextIDs []string `json:"repository_context_ids"`
	}{targetProjectID, repositoryContextIDs}
	_, err := client.Post("/api/shiftleft/repository_contexts/move_project/", body)
	return err
}

type scmRepositoryDescriptor struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
}

type scmRepoIntegrateCommon struct {
	InstallationID        string                   `json:"installation_id"`
	ConfigurationSettings ScmRepoIntegrationConfig `json:"configuration_settings"`
	ProjectID             string                   `json:"project_id,omitempty"`
}

func newScmRepoIntegrateCommon(installationID, projectID string, cfg ScmRepoIntegrationConfig) scmRepoIntegrateCommon {
	return scmRepoIntegrateCommon{InstallationID: installationID, ConfigurationSettings: cfg, ProjectID: projectID}
}

func projectID(ref *scmIDRef) string {
	if ref == nil {
		return ""
	}
	return ref.ID
}

// Shared list fields; Bitbucket nests config under configuration_settings, Azure under managed_repo_properties.
type scmRepoCommonFields struct {
	ID                  string     `json:"id"`
	Project             *scmIDRef  `json:"project"`
	Repository          scmRepoRef `json:"repository"`
	Status              string     `json:"status"`
	RepositoryContextID string     `json:"repository_context_id"`
	IntegrationStatus   string     `json:"integration_status"`
	ScmPosturePolicyID  string     `json:"scm_posture_policy_id"`

	Disabled                bool   `json:"disabled"`
	DisableScanPullRequests *bool  `json:"disable_scan_pull_requests"`
	CommentsOnPullRequests  string `json:"comments_on_pull_requests"`
	PrSummaryComment        string `json:"pr_summary_comment"`
	SkipCheckRuns           string `json:"skip_check_runs"`
	ConfigFileSupport       string `json:"config_file_support"`
}

func (r *scmRepoCommonFields) common() ScmRepository {
	return ScmRepository{
		ID:                  r.ID,
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

type githubRepositoryItem struct {
	scmRepoCommonFields
	GithubRepositoryID int64    `json:"github_repository_id"`
	GithubInstallation scmIDRef `json:"github_installation"`
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

// repositoryName is a hint-only search filter; caller still matches exactly.
func githubRepositoryNameFilter(repositoryName string) listFilters {
	return listFilters{"search": repositoryName, "search_fields": "repository_name"}
}

// github_repository_id/github_installation_id are not safe list filters; match locally. The list API
// (GithubRepositoryFilter server-side) has no github_repository_id filter at all, so the name search is
// only a cheap first pass: the repository is identified by github_repository_id, which is stable across
// renames. A filtered miss therefore means "the name is stale", not "the repository is gone", so it must
// fall back to the unfiltered scan — reporting absence would make Terraform drop a live integration from
// state and then try to re-integrate an already-integrated repository.
func (client *APIClient) FindGithubRepository(installationID, repositoryName string, githubRepositoryID int64) (*ScmRepository, error) {
	match := func(r *githubRepositoryItem) bool {
		return r.GithubInstallation.ID == installationID && r.GithubRepositoryID == githubRepositoryID
	}
	if repositoryName != "" {
		repo, err := findScmRepository(client, "github",
			githubRepositoryNameFilter(repositoryName), match, (*githubRepositoryItem).common)
		if err != nil || repo != nil {
			return repo, err
		}
	}
	return findScmRepository(client, "github", nil, match, (*githubRepositoryItem).common)
}

func (client *APIClient) UpdateGithubRepositories(body ScmRepositoryConfigUpdate) error {
	return client.updateScmRepositories("github", body)
}

type gitlabRepositoryItem struct {
	scmRepoCommonFields
	GitlabProjectID    int64    `json:"gitlab_project_id"`
	GitlabInstallation scmIDRef `json:"gitlab_installation"`
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
	return findScmRepository(client, "gitlab",
		listFilters{
			"installation_id":   installationID,
			"gitlab_project_id": strconv.FormatInt(gitlabProjectID, 10),
		},
		func(r *gitlabRepositoryItem) bool {
			return r.GitlabInstallation.ID == installationID && r.GitlabProjectID == gitlabProjectID
		},
		(*gitlabRepositoryItem).common)
}

func (client *APIClient) UpdateGitlabRepositories(body ScmRepositoryConfigUpdate) error {
	return client.updateScmRepositories("gitlab", body)
}

type bitbucketRepositoryItem struct {
	scmRepoCommonFields
	BitbucketRepositoryID string `json:"bitbucket_repository_id"`
	BitbucketRepoSlug     string `json:"bitbucket_repository_slug"`
	AccountInstallation   struct {
		ID        string `json:"id"`
		AccountID string `json:"account_id"`
	} `json:"account_installation"`
	ConfigurationSettings struct {
		DisableScanPullRequests *bool  `json:"disable_scan_pull_requests"`
		CommentsOnPullRequests  string `json:"comments_on_pull_requests"`
		PrSummaryComment        string `json:"pr_summary_comment"`
		SkipCheckRuns           string `json:"skip_check_runs"`
		ConfigFileSupport       string `json:"config_file_support"`
	} `json:"configuration_settings"`
}

func (r *bitbucketRepositoryItem) common() ScmRepository {
	c := r.scmRepoCommonFields.common()
	c.DisableScanPRs = r.ConfigurationSettings.DisableScanPullRequests
	c.CommentsOnPRs = r.ConfigurationSettings.CommentsOnPullRequests
	c.PrSummaryComment = r.ConfigurationSettings.PrSummaryComment
	c.SkipCheckRuns = r.ConfigurationSettings.SkipCheckRuns
	c.ConfigFileSupport = r.ConfigurationSettings.ConfigFileSupport
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

// Scope by Orca account-installation id, not slug — same slug can exist under multiple installations.
func (client *APIClient) FindBitbucketRepository(installationID, accountSlug, bitbucketRepositoryID string) (*ScmRepository, error) {
	account, err := client.FindBitbucketAccountBySlug(installationID, accountSlug)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errAccountLookupFailed("Bitbucket", "account/workspace", accountSlug, installationID)
	}
	return findScmRepository(client, "bitbucket",
		listFilters{"account_installation_id": account.ID},
		func(r *bitbucketRepositoryItem) bool {
			return r.AccountInstallation.ID == account.ID && r.BitbucketRepositoryID == bitbucketRepositoryID
		},
		(*bitbucketRepositoryItem).common)
}

func (client *APIClient) UpdateBitbucketRepositories(body ScmRepositoryConfigUpdate) error {
	return client.updateScmRepositories("bitbucket", body)
}

type azureRepositoryItem struct {
	scmRepoCommonFields
	AzureRepositoryID        string `json:"azure_repository_id"`
	AzureAccountInstallation struct {
		ID          string `json:"id"`
		AccountName string `json:"account_name"`
	} `json:"azure_account_installation"`
	ManagedRepoProperties struct {
		Disabled          bool   `json:"disabled"`
		ConfigFileSupport string `json:"config_file_support"`
	} `json:"managed_repo_properties"`
}

// Azure nests disabled/config in managed_repo_properties; list omits skip_check_runs.
func (r *azureRepositoryItem) common() ScmRepository {
	c := r.scmRepoCommonFields.common()
	c.Disabled = r.ManagedRepoProperties.Disabled
	c.ConfigFileSupport = r.ManagedRepoProperties.ConfigFileSupport
	return c
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

// Scope by Orca account-installation id — account names are unique per installation only.
func (client *APIClient) FindAzureRepository(installationID, accountName, azureRepositoryID string) (*ScmRepository, error) {
	account, err := client.FindAzureDevopsAccountByName(installationID, accountName)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errAccountLookupFailed("Azure DevOps", "account/organization", accountName, installationID)
	}
	return findScmRepository(client, "azure_devops",
		listFilters{
			"azure_account_installation_id": account.ID,
			"azure_repository_id":           azureRepositoryID,
		},
		func(r *azureRepositoryItem) bool {
			return r.AzureAccountInstallation.ID == account.ID && r.AzureRepositoryID == azureRepositoryID
		},
		(*azureRepositoryItem).common)
}

func (client *APIClient) UpdateAzureRepositories(body ScmRepositoryConfigUpdate) error {
	return client.updateScmRepositories("azure_devops", body)
}
