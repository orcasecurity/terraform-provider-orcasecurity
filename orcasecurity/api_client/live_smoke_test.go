package api_client

import (
	"fmt"
	"os"
	"testing"
)

// Read-only breadth check against a live tenant: it walks the shift-left list and
// lookup paths and asserts their shape, not the tenant's contents, so it stays
// valid as lab data changes. The per-resource acceptance tests cover writes.
func TestAccLiveSmoke_ShiftLeftReadPaths(t *testing.T) {
	client := liveSmokeClient(t)

	t.Run("gitlab_installations", func(t *testing.T) { smokeGitlabInstallations(t, client) })
	t.Run("bitbucket_installations", func(t *testing.T) { smokeBitbucketInstallations(t, client) })
	t.Run("azure_installations", func(t *testing.T) { smokeAzureInstallations(t, client) })

	t.Run("github_repositories", func(t *testing.T) {
		smokeRepoRows(t, client, "github", (*githubRepositoryItem).common, func(r *githubRepositoryItem) string {
			c := r.common()
			return fmt.Sprintf("gh row id=%s inst=%s ghid=%d name=%q proj=%s ctx=%s status=%s disabled=%v comments=%q",
				c.ID, r.GithubInstallation.ID, r.GithubRepositoryID, c.RepositoryName, c.ProjectID, c.RepositoryContextID, c.Status, c.Disabled, c.CommentsOnPRs)
		})
	})
	t.Run("gitlab_repositories", func(t *testing.T) {
		smokeRepoRows(t, client, "gitlab", (*gitlabRepositoryItem).common, func(r *gitlabRepositoryItem) string {
			c := r.common()
			return fmt.Sprintf("gl row id=%s inst=%s glid=%d name=%q proj=%s ctx=%s disabled=%v",
				c.ID, r.GitlabInstallation.ID, r.GitlabProjectID, c.RepositoryName, c.ProjectID, c.RepositoryContextID, c.Disabled)
		})
	})
	t.Run("bitbucket_repositories", func(t *testing.T) {
		smokeRepoRows(t, client, "bitbucket", (*bitbucketRepositoryItem).common, func(r *bitbucketRepositoryItem) string {
			c := r.common()
			return fmt.Sprintf("bb row id=%s acct=%s(%s) bbid=%s name=%q ctx=%s disabled=%v cfs=%q",
				c.ID, r.AccountInstallation.ID, r.AccountInstallation.AccountID, r.BitbucketRepositoryID, c.RepositoryName, c.RepositoryContextID, c.Disabled, c.ConfigFileSupport)
		})
	})
	t.Run("azure_repositories", func(t *testing.T) {
		smokeRepoRows(t, client, "azure_devops", (*azureRepositoryItem).common, func(r *azureRepositoryItem) string {
			c := r.common()
			return fmt.Sprintf("az row id=%s acct=%s(%q) azid=%s name=%q ctx=%s disabled=%v cfs=%q",
				c.ID, r.AzureAccountInstallation.ID, r.AzureAccountInstallation.AccountName, r.AzureRepositoryID, c.RepositoryName, c.RepositoryContextID, c.Disabled, c.ConfigFileSupport)
		})
	})

	t.Run("scm_posture_default", func(t *testing.T) { smokeScmPostureDefault(t, client) })

	// ListAutomationsV2 is the only paginateOffset caller outside the SCM lists.
	// The priority_order resource covers it live, but the automations data source
	// has no acceptance test, so this is its only live paging check.
	t.Run("automations_paging", func(t *testing.T) { smokeAutomationsPaging(t, client) })

	// Find* narrows the list server-side. The API ignores filter keys it does not
	// recognise, so a renamed or dropped filter would silently stop narrowing
	// rather than fail: these round-trips assert the filters still resolve a
	// known row, and that a mismatched installation still resolves to nothing.
	t.Run("find_by_filter_roundtrip", func(t *testing.T) {
		t.Run("github", func(t *testing.T) {
			row := firstRepoRow[githubRepositoryItem](t, client, "github")
			found, err := client.FindGithubRepository(row.GithubInstallation.ID, row.Repository.Name, row.GithubRepositoryID)
			assertFound(t, row.Repository.Name, found, err)
			// The name is a hint only: github_repository_id identifies the row and
			// survives a rename, so a stale name must still resolve through the
			// unfiltered fallback rather than read as "not found" and drop a live
			// integration from state. Simulating a rename with a name that matches
			// nothing also exercises the fallback exactly as an empty name (post-
			// import, no hint at all) does.
			renamed, err := client.FindGithubRepository(row.GithubInstallation.ID, "orca-no-such-repository", row.GithubRepositoryID)
			assertFound(t, row.Repository.Name, renamed, err)
			found, err = client.FindGithubRepository(row.GithubInstallation.ID, "", row.GithubRepositoryID)
			assertFound(t, row.Repository.Name, found, err)
			// A repository id that exists under no installation is genuinely absent.
			missing, err := client.FindGithubRepository(row.GithubInstallation.ID, row.Repository.Name, -1)
			assertNotFound(t, "github/unknown-repository-id", missing, err)
			other, err := client.FindGithubRepository(mismatchedUUID, row.Repository.Name, row.GithubRepositoryID)
			assertNotFound(t, "github/wrong-installation", other, err)
		})
		t.Run("gitlab", func(t *testing.T) {
			row := firstRepoRow[gitlabRepositoryItem](t, client, "gitlab")
			found, err := client.FindGitlabRepository(row.GitlabInstallation.ID, row.GitlabProjectID)
			assertFound(t, row.Repository.Name, found, err)
			other, err := client.FindGitlabRepository(mismatchedUUID, row.GitlabProjectID)
			assertNotFound(t, "gitlab/wrong-installation", other, err)
		})
		t.Run("bitbucket", func(t *testing.T) {
			row := firstRepoRow[bitbucketRepositoryItem](t, client, "bitbucket")
			// FindBitbucketRepository resolves the workspace by slug under an
			// installation, so drive it the way the resource does.
			accounts, err := client.ListBitbucketAccounts()
			if err != nil {
				t.Fatalf("bitbucket accounts: %v", err)
			}
			var installationID string
			for _, a := range accounts {
				if a.ID == row.AccountInstallation.ID {
					installationID = a.InstallationID
					break
				}
			}
			if installationID == "" {
				t.Skipf("no bitbucket installation found for account installation %s", row.AccountInstallation.ID)
			}
			slug := row.AccountInstallation.AccountID
			found, err := client.FindBitbucketRepository(installationID, slug, row.BitbucketRepositoryID)
			assertFound(t, row.Repository.Name, found, err)
			// A repository id the workspace does not contain is genuinely absent.
			missing, err := client.FindBitbucketRepository(installationID, slug, "orca-no-such-repository-id")
			assertNotFound(t, "bitbucket/unknown-repository-id", missing, err)
			// An unknown workspace must fail closed (account lookup error), not
			// read as absence and drop a live integration from state.
			if _, err := client.FindBitbucketRepository(installationID, "orca-no-such-workspace", row.BitbucketRepositoryID); err == nil {
				t.Error("bitbucket/wrong-workspace: expected an account-lookup error, got none")
			}
		})
		t.Run("azure_devops", func(t *testing.T) {
			row := firstRepoRow[azureRepositoryItem](t, client, "azure_devops")
			// FindAzureRepository resolves the account by name under an
			// installation, so drive it the way the resource does.
			accounts, err := client.ListAzureDevopsAccounts()
			if err != nil {
				t.Fatalf("azure accounts: %v", err)
			}
			var installationID string
			for _, a := range accounts {
				if a.ID == row.AzureAccountInstallation.ID {
					installationID = a.InstallationID
					break
				}
			}
			if installationID == "" {
				t.Skipf("no azure installation found for account installation %s", row.AzureAccountInstallation.ID)
			}
			found, err := client.FindAzureRepository(
				installationID, row.AzureAccountInstallation.AccountName, row.AzureRepositoryID)
			assertFound(t, row.Repository.Name, found, err)
		})
	})
}

// A syntactically valid UUID that will not match any real installation.
const mismatchedUUID = "00000000-0000-4000-8000-000000000000"

func firstRepoRow[T any](t *testing.T, client *APIClient, provider string) *T {
	t.Helper()
	rows, err := getAllScmPages[T](client, integratedRepositoriesPath(provider), nil)
	if err != nil {
		t.Fatalf("%s repos: %v", provider, err)
	}
	if len(rows) == 0 {
		t.Skipf("no integrated %s repositories in this tenant", provider)
	}
	return &rows[0]
}

func assertFound(t *testing.T, wantName string, got *ScmRepository, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("filtered lookup failed: %v", err)
	}
	if got == nil {
		t.Fatalf("filtered lookup found nothing; expected %q", wantName)
	}
	if got.RepositoryName != wantName {
		t.Errorf("filtered lookup returned %q, want %q", got.RepositoryName, wantName)
	}
}

func assertNotFound(t *testing.T, what string, got *ScmRepository, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", what, err)
	}
	if got != nil {
		t.Errorf("%s: expected no match, got %s (%s)", what, got.ID, got.RepositoryName)
	}
}

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
		if g.ID == "" {
			t.Errorf("gitlab installation with empty id: name=%q", g.Name)
		}
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
		if b.ID == "" {
			t.Errorf("bitbucket installation with empty id: name=%q", b.Name)
		}
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
		if a.ID == "" {
			t.Errorf("azure installation with empty id: name=%q", a.Name)
		}
		t.Logf("  azure id=%s name=%q server=%s type=%q account=%q cloud=%v",
			a.ID, a.Name, a.ServerURL, a.AccessTokenType, a.AccessTokenAccountName, a.CloudIntegration)
	}
}

func smokeRepoRows[T any](t *testing.T, client *APIClient, provider string, toCommon func(*T) ScmRepository, describe func(*T) string) {
	rows, err := getAllScmPages[T](client, integratedRepositoriesPath(provider), nil)
	if err != nil {
		t.Fatalf("%s repos: %v", provider, err)
	}
	t.Logf("%s integrated repos: %d", provider, len(rows))
	for i := range rows {
		c := toCommon(&rows[i])
		if c.ID == "" {
			t.Errorf("%s row %d: empty id", provider, i)
		}
		if c.RepositoryContextID == "" {
			t.Errorf("%s row %d (name=%q id=%s): empty repository_context_id", provider, i, c.RepositoryName, c.ID)
		}
	}
	for i := range rows {
		if i >= 3 {
			break
		}
		t.Logf("  %s", describe(&rows[i]))
	}
}

func smokeAutomationsPaging(t *testing.T, client *APIClient) {
	automations, err := client.ListAutomationsV2()
	if err != nil {
		t.Fatalf("list automations: %v", err)
	}
	t.Logf("automations: %d (page limit 300)", len(automations))

	// If start_at_index were dropped or renamed, the server would keep serving
	// page one and paging would either repeat rows or spin to the page cap. A
	// duplicate id is the observable symptom, and is worth checking even on a
	// single-page tenant since it also catches a bogus total_items.
	seen := make(map[string]bool, len(automations))
	for _, a := range automations {
		if a.ID == "" {
			t.Errorf("automation with empty id: name=%q", a.Name)
			continue
		}
		if seen[a.ID] {
			t.Errorf("automation %s returned twice: paging is re-reading a page", a.ID)
		}
		seen[a.ID] = true
	}
	if len(automations) <= 300 {
		t.Logf("  single page: multi-page paging not exercised in this tenant")
	}
}

func smokeScmPostureDefault(t *testing.T, client *APIClient) {
	pol, err := client.GetScmPostureDefaultPolicy()
	if err != nil {
		t.Fatalf("scm posture default: %v", err)
	}
	if pol.ID == "" {
		t.Errorf("scm posture default: empty id")
	}
	if pol.Name == "" {
		t.Errorf("scm posture default: empty name")
	}
	t.Logf("scm posture default: id=%s name=%q disabled=%v policy_data=%s", pol.ID, pol.Name, pol.Disabled, string(pol.PolicyData))
}
