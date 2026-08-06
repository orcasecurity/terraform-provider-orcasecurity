package api_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListGithubRepositories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/shiftleft/github/integrated_repositories/" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_items": 1,
			"data": []map[string]any{{
				"id": "repo-1", "github_repository_id": 42,
				"repository":            map[string]any{"name": "org/repo", "url": "https://github.com/org/repo"},
				"github_installation":   map[string]any{"id": "acct-1"},
				"repository_context_id": "ctx-1",
			}},
		})
	}))
	t.Cleanup(srv.Close)

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	rows, err := client.ListGithubRepositories()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].AccountID != "acct-1" || rows[0].GithubRepositoryID != 42 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestListGitlabRepositoriesStampsGitlabGroupID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/shiftleft/gitlab/installations/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data":        []map[string]any{{"id": "inst-1", "name": "GL"}},
			})
		case "/api/shiftleft/gitlab/installations/inst-1/integrated_groups/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data": []map[string]any{{
					"id": "group-orca-1", "gitlab_group_id": 133143428, "gitlab_group_name": "acme",
				}},
			})
		case "/api/shiftleft/gitlab/integrated_repositories/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data": []map[string]any{{
					"id": "repo-1", "gitlab_project_id": 99,
					"repository":            map[string]any{"name": "acme/proj", "url": "https://gitlab.com/acme/proj"},
					"gitlab_installation":   map[string]any{"id": "inst-1"},
					"gitlab_group":          map[string]any{"id": "group-orca-1", "gitlab_group_name": "acme"},
					"repository_context_id": "ctx-1",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	rows, err := client.ListGitlabRepositories()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %+v", rows)
	}
	if rows[0].InstallationID != "inst-1" || rows[0].GitlabGroupID != 133143428 || rows[0].GitlabProjectID != 99 {
		t.Fatalf("stamp/identity mismatch: %+v", rows[0])
	}
}

func TestListGitlabRepositoriesMissingGroupErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/shiftleft/gitlab/installations/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data":        []map[string]any{{"id": "inst-1", "name": "GL"}},
			})
		case "/api/shiftleft/gitlab/installations/inst-1/integrated_groups/":
			_ = json.NewEncoder(w).Encode(map[string]any{"total_items": 0, "data": []map[string]any{}})
		case "/api/shiftleft/gitlab/integrated_repositories/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data": []map[string]any{{
					"id": "repo-1", "gitlab_project_id": 99,
					"repository":            map[string]any{"name": "acme/proj", "url": "https://gitlab.com/acme/proj"},
					"gitlab_installation":   map[string]any{"id": "inst-1"},
					"gitlab_group":          map[string]any{"id": "group-orca-gone", "gitlab_group_name": "acme"},
					"repository_context_id": "ctx-1",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	if _, err := client.ListGitlabRepositories(); err == nil {
		t.Fatal("expected error for repository referencing a missing group, got nil")
	}
}

func TestListBitbucketRepositoriesStampsInstallationID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/shiftleft/bitbucket/installations/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data":        []map[string]any{{"id": "inst-1", "name": "BB"}},
			})
		case "/api/shiftleft/bitbucket/installations/inst-1/integrated_accounts/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data": []map[string]any{{
					"id": "acct-orca-1", "account_id": "ws", "account_name": "ws",
				}},
			})
		case "/api/shiftleft/bitbucket/integrated_repositories/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data": []map[string]any{{
					"id": "repo-1", "bitbucket_repository_id": "bb-1",
					"bitbucket_repository_slug": "repo",
					"repository":                map[string]any{"name": "ws/repo", "url": "https://bitbucket.org/ws/repo"},
					"account_installation":      map[string]any{"id": "acct-orca-1", "account_id": "ws"},
					"configuration_settings":    map[string]any{},
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	rows, err := client.ListBitbucketRepositories()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %+v", rows)
	}
	if rows[0].InstallationID != "inst-1" || rows[0].AccountID != "ws" || rows[0].BitbucketRepositoryID != "bb-1" {
		t.Fatalf("stamp/identity mismatch: %+v", rows[0])
	}
}

func TestListBitbucketRepositoriesMissingAccountErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/shiftleft/bitbucket/installations/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data":        []map[string]any{{"id": "inst-1", "name": "BB"}},
			})
		case "/api/shiftleft/bitbucket/installations/inst-1/integrated_accounts/":
			_ = json.NewEncoder(w).Encode(map[string]any{"total_items": 0, "data": []map[string]any{}})
		case "/api/shiftleft/bitbucket/integrated_repositories/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data": []map[string]any{{
					"id": "repo-1", "bitbucket_repository_id": "bb-1",
					"bitbucket_repository_slug": "repo",
					"repository":                map[string]any{"name": "ws/repo", "url": "https://bitbucket.org/ws/repo"},
					"account_installation":      map[string]any{"id": "acct-orca-gone", "account_id": "ws"},
					"configuration_settings":    map[string]any{},
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	if _, err := client.ListBitbucketRepositories(); err == nil {
		t.Fatal("expected error for repository referencing a missing account, got nil")
	}
}

func TestListAzureRepositoriesStampsInstallationID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/shiftleft/azure_devops/installations/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data":        []map[string]any{{"id": "inst-1", "name": "ADO"}},
			})
		case "/api/shiftleft/azure_devops/installations/inst-1/integrated_accounts/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data":        []map[string]any{{"id": "acct-orca-1", "account_name": "org-name"}},
			})
		case "/api/shiftleft/azure_devops/integrated_repositories/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data": []map[string]any{{
					"id": "repo-1", "azure_repository_id": "az-1",
					"repository":                 map[string]any{"name": "proj/repo", "url": "https://dev.azure.com/org/proj/_git/repo"},
					"azure_account_installation": map[string]any{"id": "acct-orca-1", "account_name": "org-name"},
					"repository_context_id":      "ctx-1",
				}},
			})
		case "/api/shiftleft/azure_devops/installations/inst-1/accounts/org-name/repositories/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data": []map[string]any{{
					"repository_id": "az-1", "azure_project_id": "proj-uuid-1",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	rows, err := client.ListAzureRepositories()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %+v", rows)
	}
	if rows[0].InstallationID != "inst-1" || rows[0].AccountName != "org-name" || rows[0].AzureRepositoryID != "az-1" {
		t.Fatalf("stamp/identity mismatch: %+v", rows[0])
	}
	if rows[0].AzureProjectID != "proj-uuid-1" {
		t.Fatalf("expected azure_project_id joined from browse endpoint, got %+v", rows[0])
	}
}

func TestListAzureRepositoriesMissingAccountErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/shiftleft/azure_devops/installations/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data":        []map[string]any{{"id": "inst-1", "name": "ADO"}},
			})
		case "/api/shiftleft/azure_devops/installations/inst-1/integrated_accounts/":
			_ = json.NewEncoder(w).Encode(map[string]any{"total_items": 0, "data": []map[string]any{}})
		case "/api/shiftleft/azure_devops/integrated_repositories/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data": []map[string]any{{
					"id": "repo-1", "azure_repository_id": "az-1",
					"repository":                 map[string]any{"name": "proj/repo", "url": "https://dev.azure.com/org/proj/_git/repo"},
					"azure_account_installation": map[string]any{"id": "acct-orca-gone", "account_name": "org-name"},
					"repository_context_id":      "ctx-1",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	if _, err := client.ListAzureRepositories(); err == nil {
		t.Fatal("expected error for repository referencing a missing account, got nil")
	}
}
