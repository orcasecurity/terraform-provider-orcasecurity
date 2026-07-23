package api_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func captureServer(t *testing.T, responses map[string]string) (*APIClient, *struct {
	Method, Path string
	Body         map[string]any
}) {
	t.Helper()
	last := &struct {
		Method, Path string
		Body         map[string]any
	}{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last.Method = r.Method
		last.Path = r.URL.Path
		last.Body = nil
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&last.Body)
		}
		if resp, ok := responses[r.Method+" "+r.URL.Path]; ok {
			_, _ = w.Write([]byte(resp))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}, last
}

func TestIntegrateGithubRepository_BodyShape(t *testing.T) {
	client, last := captureServer(t, nil)
	err := client.IntegrateGithubRepository(GithubRepositoryIntegrate{
		InstallationID:     "inst-1",
		GithubRepositoryID: 42,
		Name:               "acme/repo",
		URL:                "https://github.com/acme/repo",
		Branch:             "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if last.Path != "/api/shiftleft/github/integrated_repositories/" || last.Method != "POST" {
		t.Fatalf("wrong request: %s %s", last.Method, last.Path)
	}
	if last.Body["installation_id"] != "inst-1" {
		t.Errorf("installation_id: %v", last.Body["installation_id"])
	}
	if _, ok := last.Body["configuration_settings"]; !ok {
		t.Error("configuration_settings missing")
	}
	if _, ok := last.Body["project_id"]; ok {
		t.Error("empty project_id must be omitted")
	}
	repos := last.Body["repositories"].([]any)
	repo := repos[0].(map[string]any)
	if repo["github_repository_id"] != float64(42) || repo["name"] != "acme/repo" || repo["branch"] != "main" {
		t.Errorf("bad repository entry: %v", repo)
	}
}

func TestIntegrateGitlabRepository_BodyShape(t *testing.T) {
	client, last := captureServer(t, nil)
	err := client.IntegrateGitlabRepository(GitlabRepositoryIntegrate{
		InstallationID:  "inst-1",
		GitlabGroupID:   7,
		GitlabProjectID: 99,
		Name:            "grp/proj",
		URL:             "https://gitlab.com/grp/proj",
		ProjectID:       "proj-uuid",
	})
	if err != nil {
		t.Fatal(err)
	}
	if last.Path != "/api/shiftleft/gitlab/integrated_repositories/" {
		t.Fatalf("wrong path: %s", last.Path)
	}
	if last.Body["group_id"] != float64(7) || last.Body["project_id"] != "proj-uuid" {
		t.Errorf("bad top-level: %v", last.Body)
	}
	repo := last.Body["repositories"].([]any)[0].(map[string]any)
	if repo["id"] != float64(99) {
		t.Errorf("gitlab repo id must marshal as \"id\": %v", repo)
	}
	if _, ok := repo["branch"]; ok {
		t.Error("empty branch must be omitted")
	}
}

func TestIntegrateBitbucketRepository_BodyShape(t *testing.T) {
	client, last := captureServer(t, nil)
	err := client.IntegrateBitbucketRepository(BitbucketRepositoryIntegrate{
		InstallationID:        "inst-1",
		AccountID:             "workspace-slug",
		BitbucketRepositoryID: "repo-id",
		Slug:                  "repo-slug",
		Name:                  "repo",
		URL:                   "https://bitbucket.org/w/repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if last.Path != "/api/shiftleft/bitbucket/integrated_repositories/" {
		t.Fatalf("wrong path: %s", last.Path)
	}
	if last.Body["account_id"] != "workspace-slug" {
		t.Errorf("account_id: %v", last.Body["account_id"])
	}
	repo := last.Body["repositories"].([]any)[0].(map[string]any)
	if repo["id"] != "repo-id" || repo["slug"] != "repo-slug" {
		t.Errorf("bad repository entry: %v", repo)
	}
}

func TestIntegrateAzureRepository_BodyShape(t *testing.T) {
	client, last := captureServer(t, nil)
	err := client.IntegrateAzureRepository(AzureRepositoryIntegrate{
		InstallationID:    "inst-1",
		AccountName:       "org-name",
		AzureRepositoryID: "repo-uuid",
		AzureProjectID:    "azproj-uuid",
		Name:              "repo",
		URL:               "https://dev.azure.com/org/proj/_git/repo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if last.Path != "/api/shiftleft/azure_devops/integrated_repositories/" {
		t.Fatalf("wrong path: %s", last.Path)
	}
	if last.Body["azure_account_name"] != "org-name" {
		t.Errorf("azure_account_name: %v", last.Body["azure_account_name"])
	}
	repo := last.Body["repositories"].([]any)[0].(map[string]any)
	if repo["id"] != "repo-uuid" || repo["azure_project_id"] != "azproj-uuid" {
		t.Errorf("bad repository entry: %v", repo)
	}
}

func TestScmRepositoryConfigUpdate_MarshalOmitsUnset(t *testing.T) {
	f := false
	raw, _ := json.Marshal(ScmRepositoryConfigUpdate{
		IDs:      []string{"row-1"},
		Disabled: &f,
	})
	var got map[string]any
	_ = json.Unmarshal(raw, &got)
	if got["disabled"] != false {
		t.Errorf("explicit false disabled must be sent: %s", raw)
	}
	for _, k := range []string{"disable_scan_pull_requests", "comments_on_pull_requests", "pr_summary_comment", "skip_check_runs", "config_file_support"} {
		if _, ok := got[k]; ok {
			t.Errorf("unset %q must be omitted (null means clear nothing, omit means leave unchanged): %s", k, raw)
		}
	}
}

func TestFindGithubRepository_NormalizesFlatItem(t *testing.T) {
	list := `{"total_items":1,"data":[{
		"id":"row-1","github_repository_id":42,
		"github_installation":{"id":"inst-1"},
		"project":{"id":"proj-1"},
		"repository":{"name":"acme/repo","url":"https://github.com/acme/repo"},
		"disabled":false,"disable_scan_pull_requests":true,
		"comments_on_pull_requests":"NEVER","pr_summary_comment":"ALWAYS",
		"skip_check_runs":"ALWAYS","config_file_support":"ENABLED",
		"status":"SUCCESS","repository_context_id":"ctx-1",
		"integration_status":null,"scm_posture_policy_id":"scm-pol-1"}]}`
	client, _ := captureServer(t, map[string]string{
		"GET /api/shiftleft/github/integrated_repositories/": list,
	})
	row, err := client.FindGithubRepository("inst-1", 42)
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("row not found")
	}
	if row.UnitID != "inst-1" || row.ProjectID != "proj-1" || row.RepositoryContextID != "ctx-1" {
		t.Errorf("bad normalization: %+v", row)
	}
	if row.DisableScanPRs == nil || !*row.DisableScanPRs || row.CommentsOnPRs != "NEVER" {
		t.Errorf("bad config normalization: %+v", row)
	}
	client.invalidateScmListCache()
	other, err := client.FindGithubRepository("inst-other", 42)
	if err != nil || other != nil {
		t.Errorf("expected no match for other installation, got %+v (%v)", other, err)
	}
}

func TestFindBitbucketRepository_NormalizesNestedConfig(t *testing.T) {
	list := `{"total_items":1,"data":[{
		"id":"row-1","bitbucket_repository_id":"bb-1","bitbucket_repository_slug":"repo",
		"account_installation":{"id":"acct-row","account_id":"workspace-slug","account_name":"WS"},
		"project":null,
		"repository":{"name":"repo","url":"https://bitbucket.org/w/repo"},
		"disabled":true,
		"configuration_settings":{"disable_scan_pull_requests":null,"comments_on_pull_requests":"ALWAYS","pr_summary_comment":null,"skip_check_runs":null,"config_file_support":"DISABLED"},
		"status":"IN_PROGRESS","repository_context_id":"ctx-9","integration_status":null}]}`
	client, _ := captureServer(t, map[string]string{
		"GET /api/shiftleft/bitbucket/integrated_repositories/": list,
	})
	row, err := client.FindBitbucketRepository("workspace-slug", "bb-1")
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("row not found")
	}
	if !row.Disabled || row.CommentsOnPRs != "ALWAYS" || row.ConfigFileSupport != "DISABLED" {
		t.Errorf("nested configuration_settings not flattened: %+v", row)
	}
	if row.DisableScanPRs != nil {
		t.Errorf("null override must stay nil: %+v", row.DisableScanPRs)
	}
}

func TestFindAzureRepository_NormalizesManagedProperties(t *testing.T) {
	list := `{"total_items":1,"data":[{
		"id":"row-1","azure_repository_id":"az-repo-uuid",
		"azure_account_installation":{"id":"acct-row","account_name":"org-name"},
		"project":{"id":"proj-2"},
		"repository":{"name":"repo","url":"https://dev.azure.com/org/proj/_git/repo"},
		"managed_repo_properties":{"disabled":true,"config_file_support":"ENABLED"},
		"disable_scan_pull_requests":false,"comments_on_pull_requests":"NEVER","pr_summary_comment":"NEVER",
		"status":"FAILED","repository_context_id":"ctx-2","integration_status":"INSTALLATION_UNREACHABLE",
		"scm_posture_policy_id":null}]}`
	client, _ := captureServer(t, map[string]string{
		"GET /api/shiftleft/azure_devops/integrated_repositories/": list,
	})
	row, err := client.FindAzureRepository("org-name", "az-repo-uuid")
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("row not found")
	}
	if !row.Disabled || row.ConfigFileSupport != "ENABLED" {
		t.Errorf("managed_repo_properties not flattened: %+v", row)
	}
	if row.IntegrationStatus != "INSTALLATION_UNREACHABLE" {
		t.Errorf("integration_status: %+v", row.IntegrationStatus)
	}
}

func TestDeleteRepositoryContext_Path(t *testing.T) {
	client, last := captureServer(t, nil)
	if err := client.DeleteRepositoryContext("ctx-1"); err != nil {
		t.Fatal(err)
	}
	if last.Method != "DELETE" || last.Path != "/api/shiftleft/repository_contexts/ctx-1/" {
		t.Fatalf("wrong request: %s %s", last.Method, last.Path)
	}
}

func TestMoveRepositoryContexts_BodyShape(t *testing.T) {
	client, last := captureServer(t, nil)
	if err := client.MoveRepositoryContexts("proj-1", []string{"ctx-1"}); err != nil {
		t.Fatal(err)
	}
	if last.Path != "/api/shiftleft/repository_contexts/move_project/" {
		t.Fatalf("wrong path: %s", last.Path)
	}
	if last.Body["target_project_id"] != "proj-1" {
		t.Errorf("target_project_id: %v", last.Body)
	}
	ids := last.Body["repository_context_ids"].([]any)
	if len(ids) != 1 || ids[0] != "ctx-1" {
		t.Errorf("repository_context_ids: %v", ids)
	}
}
