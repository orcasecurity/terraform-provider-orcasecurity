package api_client

import (
	"fmt"
	"os"
	"testing"
)

// Read-only live smoke check for the SCM installation / integrated
// repository / scm-posture endpoints: verifies the DTOs parse real payloads
// and that no write-only secret is ever echoed. Gated on TF_ACC like the
// acceptance tests (this is a plain Go test, so the check is explicit rather
// than via resource.Test) and uses the same ORCASECURITY_API_ENDPOINT/TOKEN
// credentials; it never mutates the org.
func TestAccLiveSmoke_NewEndpoints(t *testing.T) {
	client := liveSmokeClient(t)

	t.Run("gitlab_installations", func(t *testing.T) { smokeGitlabInstallations(t, client) })
	t.Run("bitbucket_installations", func(t *testing.T) { smokeBitbucketInstallations(t, client) })
	t.Run("azure_installations", func(t *testing.T) { smokeAzureInstallations(t, client) })

	t.Run("github_repositories", func(t *testing.T) {
		smokeRepoRows(t, client, "github", func(r *githubRepositoryItem) string {
			c := r.common()
			return fmt.Sprintf("gh row id=%s unit=%s ghid=%d name=%q proj=%s ctx=%s status=%s disabled=%v comments=%q",
				c.ID, c.UnitID, r.GithubRepositoryID, c.RepositoryName, c.ProjectID, c.RepositoryContextID, c.Status, c.Disabled, c.CommentsOnPRs)
		})
	})
	t.Run("gitlab_repositories", func(t *testing.T) {
		smokeRepoRows(t, client, "gitlab", func(r *gitlabRepositoryItem) string {
			c := r.common()
			return fmt.Sprintf("gl row id=%s inst=%s unit=%s glid=%d name=%q proj=%s ctx=%s disabled=%v",
				c.ID, r.GitlabInstallation.ID, c.UnitID, r.GitlabProjectID, c.RepositoryName, c.ProjectID, c.RepositoryContextID, c.Disabled)
		})
	})
	t.Run("bitbucket_repositories", func(t *testing.T) {
		smokeRepoRows(t, client, "bitbucket", func(r *bitbucketRepositoryItem) string {
			c := r.common()
			return fmt.Sprintf("bb row id=%s acct=%s(%s) bbid=%s name=%q ctx=%s disabled=%v cfs=%q",
				c.ID, r.AccountInstallation.ID, r.AccountInstallation.AccountID, r.BitbucketRepositoryID, c.RepositoryName, c.RepositoryContextID, c.Disabled, c.ConfigFileSupport)
		})
	})
	t.Run("azure_repositories", func(t *testing.T) {
		smokeRepoRows(t, client, "azure_devops", func(r *azureRepositoryItem) string {
			c := r.common()
			return fmt.Sprintf("az row id=%s acct=%s(%q) azid=%s name=%q ctx=%s disabled=%v cfs=%q",
				c.ID, r.AzureAccountInstallation.ID, r.AzureAccountInstallation.AccountName, r.AzureRepositoryID, c.RepositoryName, c.RepositoryContextID, c.Disabled, c.ConfigFileSupport)
		})
	})

	t.Run("scm_posture_default", func(t *testing.T) { smokeScmPostureDefault(t, client) })
}

// liveSmokeClient skips unless TF_ACC and credentials are set, then builds a
// client against the live API.
func liveSmokeClient(t *testing.T) *APIClient {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run live tests")
	}
	endpoint := os.Getenv("ORCASECURITY_API_ENDPOINT")
	token := os.Getenv("ORCASECURITY_API_TOKEN")
	if endpoint == "" || token == "" {
		t.Skip("ORCASECURITY_API_ENDPOINT / ORCASECURITY_API_TOKEN not set")
	}
	client, err := NewAPIClient(&endpoint, &token)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func smokeGitlabInstallations(t *testing.T, client *APIClient) {
	installations, err := client.ListGitlabInstallations()
	if err != nil {
		t.Fatalf("gitlab installations: %v", err)
	}
	t.Logf("gitlab installations: %d", len(installations))
	for _, g := range installations {
		t.Logf("  gitlab id=%s name=%q server=%s readonly=%v cloud=%v status=%q token_name=%q token_type=%q",
			g.ID, g.Name, g.ServerURL, g.ReadOnly, g.CloudIntegration, g.IntegrationStatus, g.AccessTokenName, g.AccessTokenType)
	}
}

func smokeBitbucketInstallations(t *testing.T, client *APIClient) {
	installations, err := client.ListBitbucketInstallations()
	if err != nil {
		t.Fatalf("bitbucket installations: %v", err)
	}
	t.Logf("bitbucket installations: %d", len(installations))
	for _, b := range installations {
		td := b.AccessTokenDetails
		if td == nil {
			td = &BitbucketAccessTokenDetails{}
		}
		t.Logf("  bitbucket id=%s name=%q server=%s cloud=%v type=%q account=%q user=%q secret_leaked=%v",
			b.ID, b.Name, b.ServerURL, b.CloudIntegration, td.AccessTokenType, td.AccountID, td.Username, td.AccessToken != "")
	}
}

func smokeAzureInstallations(t *testing.T, client *APIClient) {
	installations, err := client.ListAzureDevopsInstallations()
	if err != nil {
		t.Fatalf("azure installations: %v", err)
	}
	t.Logf("azure installations: %d", len(installations))
	for _, a := range installations {
		t.Logf("  azure id=%s name=%q server=%s type=%q account=%q cloud=%v",
			a.ID, a.Name, a.ServerURL, a.AccessTokenType, a.AccessTokenAccountName, a.CloudIntegration)
	}
}

// smokeRepoRows lists a provider's integrated repositories (validating the
// DTO against live payloads) and logs the first few rows via describe.
func smokeRepoRows[T any](t *testing.T, client *APIClient, provider string, describe func(*T) string) {
	rows, err := getAllScmPages[T](client, integratedRepositoriesPath(provider))
	if err != nil {
		t.Fatalf("%s repos: %v", provider, err)
	}
	t.Logf("%s integrated repos: %d", provider, len(rows))
	for i := range rows {
		if i >= 3 {
			break
		}
		t.Logf("  %s", describe(&rows[i]))
	}
}

func smokeScmPostureDefault(t *testing.T, client *APIClient) {
	pol, err := client.GetScmPostureDefaultPolicy()
	if err != nil {
		t.Fatalf("scm posture default: %v", err)
	}
	t.Logf("scm posture default: id=%s name=%q disabled=%v policy_data=%s", pol.ID, pol.Name, pol.Disabled, string(pol.PolicyData))
}
