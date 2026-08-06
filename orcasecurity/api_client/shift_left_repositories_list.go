package api_client

import (
	"fmt"
	"net/url"
)

type GithubRepositoryListItem struct {
	ScmRepository
	AccountID          string // Orca GitHub account UUID (= github_installation.id)
	GithubRepositoryID int64
}

type GitlabRepositoryListItem struct {
	ScmRepository
	InstallationID  string
	GitlabGroupID   int64 // numeric GitLab group id (stamped from the owning group unit)
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
	// Absent from integrated_repositories; joined from the per-account browse endpoint.
	AzureProjectID string
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
	// List rows expose gitlab_group.id (Orca unit UUID), not the numeric
	// gitlab_group_id the repository resource requires — stamp it from groups.
	groups, err := client.ListGitlabGroups()
	if err != nil {
		return nil, err
	}
	groupIDByOrcaID := make(map[string]int64, len(groups))
	for _, g := range groups {
		groupIDByOrcaID[g.ID] = g.GitlabGroupID
	}
	rows, err := getAllScmPages[gitlabRepositoryItem](client, integratedRepositoriesPath("gitlab"), nil)
	if err != nil {
		return nil, err
	}
	out := make([]GitlabRepositoryListItem, len(rows))
	for i := range rows {
		groupID, ok := groupIDByOrcaID[rows[i].GitlabGroup.ID]
		if !ok {
			return nil, fmt.Errorf(
				"gitlab repository %s references group %s which is not in the integrated groups list; "+
					"the group may have been de-integrated out-of-band", rows[i].ID, rows[i].GitlabGroup.ID)
		}
		out[i] = GitlabRepositoryListItem{
			ScmRepository:   rows[i].common(),
			InstallationID:  rows[i].GitlabInstallation.ID,
			GitlabGroupID:   groupID,
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
		installationID, ok := installByAccount[rows[i].AccountInstallation.ID]
		if !ok {
			return nil, fmt.Errorf(
				"bitbucket repository %s references account %s which is not in the integrated accounts list; "+
					"the account may have been de-integrated out-of-band", rows[i].ID, rows[i].AccountInstallation.ID)
		}
		c := rows[i].common()
		out[i] = BitbucketRepositoryListItem{
			ScmRepository:         c,
			InstallationID:        installationID,
			AccountID:             rows[i].AccountInstallation.AccountID,
			BitbucketRepositoryID: rows[i].BitbucketRepositoryID,
		}
	}
	return out, nil
}

// integrated_repositories omits azure_project_id; the per-account browse route
// (installations/{id}/accounts/{name}/repositories/) has it.
func (client *APIClient) getAzureAccountRepositoryProjectIDs(installationID, accountName string) (map[string]string, error) {
	resp, err := client.Get(fmt.Sprintf(
		"/api/shiftleft/azure_devops/installations/%s/accounts/%s/repositories/",
		installationID, url.PathEscape(accountName),
	))
	if err != nil {
		return nil, err
	}
	var body struct {
		Data []struct {
			RepositoryID   string `json:"repository_id"`
			AzureProjectID string `json:"azure_project_id"`
		} `json:"data"`
	}
	if err := resp.ReadJSON(&body); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(body.Data))
	for _, r := range body.Data {
		out[r.RepositoryID] = r.AzureProjectID
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
	out, err := buildAzureRepositoryListItems(rows, installByAccount)
	if err != nil {
		return nil, err
	}
	if err := client.joinAzureProjectIDs(out); err != nil {
		return nil, err
	}
	return out, nil
}

func buildAzureRepositoryListItems(rows []azureRepositoryItem, installByAccount map[string]string) ([]AzureRepositoryListItem, error) {
	out := make([]AzureRepositoryListItem, len(rows))
	for i := range rows {
		installationID, ok := installByAccount[rows[i].AzureAccountInstallation.ID]
		if !ok {
			return nil, fmt.Errorf(
				"azure devops repository %s references account %s which is not in the integrated accounts list; "+
					"the account may have been de-integrated out-of-band", rows[i].ID, rows[i].AzureAccountInstallation.ID)
		}
		out[i] = AzureRepositoryListItem{
			ScmRepository:     rows[i].common(),
			InstallationID:    installationID,
			AccountName:       rows[i].AzureAccountInstallation.AccountName,
			AzureRepositoryID: rows[i].AzureRepositoryID,
			AzureProjectID:    rows[i].AzureProjectID,
		}
	}
	return out, nil
}

// Fall back to the browse endpoint, once per account, only for rows the backend didn't already stamp.
func (client *APIClient) joinAzureProjectIDs(out []AzureRepositoryListItem) error {
	projectIDsByAccount := make(map[string]map[string]string)
	for i := range out {
		if out[i].AzureProjectID != "" {
			continue
		}
		accountKey := out[i].InstallationID + "/" + out[i].AccountName
		projectIDs, cached := projectIDsByAccount[accountKey]
		if !cached {
			var err error
			projectIDs, err = client.getAzureAccountRepositoryProjectIDs(out[i].InstallationID, out[i].AccountName)
			if err != nil {
				return fmt.Errorf(
					"azure devops repository %s: fetching azure_project_id from account %s failed: %w",
					out[i].AzureRepositoryID, out[i].AccountName, err)
			}
			projectIDsByAccount[accountKey] = projectIDs
		}
		projectID, ok := projectIDs[out[i].AzureRepositoryID]
		if !ok {
			return fmt.Errorf(
				"azure devops repository %s (account %s) has no azure_project_id in the account browse listing; "+
					"the repository may have been removed on the Azure DevOps side", out[i].AzureRepositoryID, out[i].AccountName)
		}
		out[i].AzureProjectID = projectID
	}
	return nil
}
