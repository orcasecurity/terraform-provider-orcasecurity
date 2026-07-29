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

// ScmRepositoryConfigUpdate is the PATCH body for integrated repositories: the
// same configuration the integrate POST carries, plus the two fields only a PATCH
// can express — the target rows and `disabled`.
//
// The embedded struct is anonymous and untagged, so encoding/json promotes its
// members: the wire body stays one flat object, unchanged from when these fields
// were declared twice. TestScmRepositoryConfigUpdate_MarshalOmitsUnset pins that.
type ScmRepositoryConfigUpdate struct {
	IDs      []string `json:"ids"`
	Disabled *bool    `json:"disabled,omitempty"`
	ScmRepoIntegrationConfig
}

func integratedRepositoriesPath(provider string) string {
	return fmt.Sprintf("/api/shiftleft/%s/integrated_repositories/", provider)
}

// findScmRepository fetches the integrated repositories a provider's filters
// narrow to, then returns the first local match.
//
// The match is still checked locally because the API ignores filter keys it does
// not recognise: filters shrink the response but cannot be trusted to have been
// applied, so they are an optimisation and never the correctness boundary.
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

func projectID(ref *scmIDRef) string {
	if ref == nil {
		return ""
	}
	return ref.ID
}

// scmRepoCommonFields are the integrated-repository fields the provider list
// serializers agree on. Each provider embeds it and declares only its own keys.
//
// The configuration block is laid out the way GitHub and GitLab serialize it:
// flat, at the top level. Bitbucket nests the same values under
// configuration_settings and Azure splits them across managed_repo_properties, so
// those two override common() to redirect the fields they place elsewhere. Keys a
// provider does not send simply stay at their zero value.
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

// GitHub serializes every shared field flat, so common() is inherited as-is.
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

// githubRepositoryNameFilter narrows the GitHub list to rows whose repository
// name matches, via the search backend (search_fields maps repository_name onto
// repository_context__repository__name). It is a substring, case-insensitive
// match, so it narrows rather than identifies — the caller still matches exactly.
func githubRepositoryNameFilter(repositoryName string) listFilters {
	return listFilters{"search": repositoryName, "search_fields": "repository_name"}
}

// FindGithubRepository locates one integrated repository by installation and
// GitHub-side repository id.
//
// Neither identifier it matches on can be pushed server-side: github_repository_id
// is not a declared filter and is silently ignored, and github_installation_id is
// derived from a foreign key and therefore choice-validated — an installation
// deleted out of band yields 400 ("Select a valid choice") rather than no rows,
// which would turn a re-creatable "not found" into a hard plan failure.
//
// repositoryName is filterable and is *not* choice-validated: a name that matches
// nothing returns an empty page. It is used as a narrowing hint when known (49
// rows down to 1 on the reference tenant). Because a stale name would otherwise
// read as a deleted repository, a miss falls back to the unfiltered scan, keeping
// the filter a pure optimisation. An empty name (the first Read after an import,
// whose id carries only installation and repository id) skips straight to it.
func (client *APIClient) FindGithubRepository(installationID, repositoryName string, githubRepositoryID int64) (*ScmRepository, error) {
	match := func(r *githubRepositoryItem) bool {
		return r.GithubInstallation.ID == installationID && r.GithubRepositoryID == githubRepositoryID
	}
	if repositoryName != "" {
		found, err := findScmRepository(client, "github",
			githubRepositoryNameFilter(repositoryName), match, (*githubRepositoryItem).common)
		if err != nil || found != nil {
			return found, err
		}
	}
	return findScmRepository(client, "github", nil, match, (*githubRepositoryItem).common)
}

func (client *APIClient) UpdateGithubRepositories(body ScmRepositoryConfigUpdate) error {
	return client.updateScmRepositories("github", body)
}

// GitLab serializes every shared field flat, so common() is inherited as-is.
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

// Bitbucket nests the configuration under configuration_settings and is the only
// provider that carries a repository slug. It sends no scm_posture_policy_id, so
// the inherited field stays empty.
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

// FindBitbucketRepository scopes the match to the parent installation's Orca
// account-installation id, not the external account slug: the same slug can be
// integrated under multiple installations in one org (slugs are unique only per
// installation, and bitbucket repository ids carry no uniqueness at all), so
// matching on the slug alone can return a repository under the wrong installation.
// bitbucket_repository_id is not a supported server-side filter, so narrow to the
// account installation and match the repository id locally.
func (client *APIClient) FindBitbucketRepository(installationID, accountSlug, bitbucketRepositoryID string) (*ScmRepository, error) {
	account, err := client.FindBitbucketAccountBySlug(installationID, accountSlug)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, nil // account not integrated under this installation
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

// Azure carries disabled and config_file_support inside managed_repo_properties
// rather than at the top level. Its list serializer also omits skip_check_runs
// entirely, which is why the resource layer keeps the prior value instead of
// treating the empty read as drift (see repoOps.skipCheckRunsUnreadable).
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
