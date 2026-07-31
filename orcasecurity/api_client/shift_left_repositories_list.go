package api_client

// List DTOs for repository data sources. Provider-specific identity fields are filled so
// for_each can feed the matching repository resource without an extra join when possible.

type GithubRepositoryListItem struct {
	ScmRepository
	AccountID          string // Orca GitHub account UUID (= github_installation.id)
	GithubRepositoryID int64
}

type GitlabRepositoryListItem struct {
	ScmRepository
	InstallationID  string
	GitlabProjectID int64
}

type BitbucketRepositoryListItem struct {
	ScmRepository
	InstallationID        string // stamped from owning BitbucketAccount
	AccountID             string // Bitbucket workspace slug / project key
	BitbucketRepositoryID string
}

type AzureRepositoryListItem struct {
	ScmRepository
	InstallationID    string // stamped from owning AzureDevopsAccount
	AccountName       string
	AzureRepositoryID string
}

func (client *APIClient) ListGithubRepositories() ([]GithubRepositoryListItem, error) {
	rows, err := getAllScmPages[githubRepositoryItem](client, integratedRepositoriesPath("github"), nil)
	if err != nil {
		return nil, err
	}
	out := make([]GithubRepositoryListItem, len(rows))
	for i := range rows {
		out[i] = GithubRepositoryListItem{
			ScmRepository:      rows[i].common(),
			AccountID:          rows[i].GithubInstallation.ID,
			GithubRepositoryID: rows[i].GithubRepositoryID,
		}
	}
	return out, nil
}

func (client *APIClient) ListGitlabRepositories() ([]GitlabRepositoryListItem, error) {
	rows, err := getAllScmPages[gitlabRepositoryItem](client, integratedRepositoriesPath("gitlab"), nil)
	if err != nil {
		return nil, err
	}
	out := make([]GitlabRepositoryListItem, len(rows))
	for i := range rows {
		out[i] = GitlabRepositoryListItem{
			ScmRepository:   rows[i].common(),
			InstallationID:  rows[i].GitlabInstallation.ID,
			GitlabProjectID: rows[i].GitlabProjectID,
		}
	}
	return out, nil
}

func (client *APIClient) ListBitbucketRepositories() ([]BitbucketRepositoryListItem, error) {
	accounts, err := client.ListBitbucketAccounts()
	if err != nil {
		return nil, err
	}
	installByAccount := make(map[string]string, len(accounts))
	for _, a := range accounts {
		installByAccount[a.ID] = a.InstallationID
	}
	rows, err := getAllScmPages[bitbucketRepositoryItem](client, integratedRepositoriesPath("bitbucket"), nil)
	if err != nil {
		return nil, err
	}
	out := make([]BitbucketRepositoryListItem, len(rows))
	for i := range rows {
		c := rows[i].common()
		out[i] = BitbucketRepositoryListItem{
			ScmRepository:         c,
			InstallationID:        installByAccount[rows[i].AccountInstallation.ID],
			AccountID:             rows[i].AccountInstallation.AccountID,
			BitbucketRepositoryID: rows[i].BitbucketRepositoryID,
		}
	}
	return out, nil
}

func (client *APIClient) ListAzureRepositories() ([]AzureRepositoryListItem, error) {
	accounts, err := client.ListAzureDevopsAccounts()
	if err != nil {
		return nil, err
	}
	installByAccount := make(map[string]string, len(accounts))
	for _, a := range accounts {
		installByAccount[a.ID] = a.InstallationID
	}
	rows, err := getAllScmPages[azureRepositoryItem](client, integratedRepositoriesPath("azure_devops"), nil)
	if err != nil {
		return nil, err
	}
	out := make([]AzureRepositoryListItem, len(rows))
	for i := range rows {
		out[i] = AzureRepositoryListItem{
			ScmRepository:     rows[i].common(),
			InstallationID:    installByAccount[rows[i].AzureAccountInstallation.ID],
			AccountName:       rows[i].AzureAccountInstallation.AccountName,
			AzureRepositoryID: rows[i].AzureRepositoryID,
		}
	}
	return out, nil
}
