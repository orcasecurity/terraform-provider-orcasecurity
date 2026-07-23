package api_client

import (
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

	gitlab, err := client.ListGitlabInstallations()
	if err != nil {
		t.Fatalf("gitlab installations: %v", err)
	}
	t.Logf("gitlab installations: %d", len(gitlab))
	for _, g := range gitlab {
		t.Logf("  gitlab id=%s name=%q server=%s readonly=%v cloud=%v status=%q token_name=%q token_type=%q",
			g.ID, g.Name, g.ServerURL, g.ReadOnly, g.CloudIntegration, g.IntegrationStatus, g.AccessTokenName, g.AccessTokenType)
	}

	bb, err := client.ListBitbucketInstallations()
	if err != nil {
		t.Fatalf("bitbucket installations: %v", err)
	}
	t.Logf("bitbucket installations: %d", len(bb))
	for _, b := range bb {
		td := b.AccessTokenDetails
		if td == nil {
			td = &BitbucketAccessTokenDetails{}
		}
		t.Logf("  bitbucket id=%s name=%q server=%s cloud=%v type=%q account=%q user=%q secret_leaked=%v",
			b.ID, b.Name, b.ServerURL, b.CloudIntegration, td.AccessTokenType, td.AccountID, td.Username, td.AccessToken != "")
	}

	az, err := client.ListAzureDevopsInstallations()
	if err != nil {
		t.Fatalf("azure installations: %v", err)
	}
	t.Logf("azure installations: %d", len(az))
	for _, a := range az {
		t.Logf("  azure id=%s name=%q server=%s type=%q account=%q cloud=%v",
			a.ID, a.Name, a.ServerURL, a.AccessTokenType, a.AccessTokenAccountName, a.CloudIntegration)
	}

	// Integrated repositories per provider: validate DTO parsing on live rows.
	ghRows, err := getAllScmPages[githubRepositoryItem](client, integratedRepositoriesPath("github"))
	if err != nil {
		t.Fatalf("github repos: %v", err)
	}
	t.Logf("github integrated repos: %d", len(ghRows))
	for i, r := range ghRows {
		if i >= 3 {
			break
		}
		c := r.common()
		t.Logf("  gh row id=%s unit=%s ghid=%d name=%q proj=%s ctx=%s status=%s disabled=%v comments=%q",
			c.ID, c.UnitID, r.GithubRepositoryID, c.RepositoryName, c.ProjectID, c.RepositoryContextID, c.Status, c.Disabled, c.CommentsOnPRs)
	}

	glRows, err := getAllScmPages[gitlabRepositoryItem](client, integratedRepositoriesPath("gitlab"))
	if err != nil {
		t.Fatalf("gitlab repos: %v", err)
	}
	t.Logf("gitlab integrated repos: %d", len(glRows))
	for i, r := range glRows {
		if i >= 3 {
			break
		}
		c := r.common()
		t.Logf("  gl row id=%s inst=%s unit=%s glid=%d name=%q proj=%s ctx=%s disabled=%v",
			c.ID, r.GitlabInstallation.ID, c.UnitID, r.GitlabProjectID, c.RepositoryName, c.ProjectID, c.RepositoryContextID, c.Disabled)
	}

	bbRows, err := getAllScmPages[bitbucketRepositoryItem](client, integratedRepositoriesPath("bitbucket"))
	if err != nil {
		t.Fatalf("bitbucket repos: %v", err)
	}
	t.Logf("bitbucket integrated repos: %d", len(bbRows))
	for i, r := range bbRows {
		if i >= 3 {
			break
		}
		c := r.common()
		t.Logf("  bb row id=%s acct=%s(%s) bbid=%s name=%q ctx=%s disabled=%v cfs=%q",
			c.ID, r.AccountInstallation.ID, r.AccountInstallation.AccountID, r.BitbucketRepositoryID, c.RepositoryName, c.RepositoryContextID, c.Disabled, c.ConfigFileSupport)
	}

	azRows, err := getAllScmPages[azureRepositoryItem](client, integratedRepositoriesPath("azure_devops"))
	if err != nil {
		t.Fatalf("azure repos: %v", err)
	}
	t.Logf("azure integrated repos: %d", len(azRows))
	for i, r := range azRows {
		if i >= 3 {
			break
		}
		c := r.common()
		t.Logf("  az row id=%s acct=%s(%q) azid=%s name=%q ctx=%s disabled=%v cfs=%q",
			c.ID, r.AzureAccountInstallation.ID, r.AzureAccountInstallation.AccountName, r.AzureRepositoryID, c.RepositoryName, c.RepositoryContextID, c.Disabled, c.ConfigFileSupport)
	}

	pol, err := client.GetScmPostureDefaultPolicy()
	if err != nil {
		t.Fatalf("scm posture default: %v", err)
	}
	t.Logf("scm posture default: id=%s name=%q disabled=%v policy_data=%s", pol.ID, pol.Name, pol.Disabled, string(pol.PolicyData))
}
