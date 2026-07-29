package api_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// Each Find*Repository must push its provider's supported filters into the query
// so a single-repository lookup does not walk the whole tenant.
func TestFindRepository_SendsServerSideFilters(t *testing.T) {
	tests := []struct {
		name string
		// row is the single list row the fake API returns.
		row    map[string]any
		find   func(*APIClient) (*ScmRepository, error)
		want   url.Values
		forbid []string
	}{
		{
			// GitHub narrows by repository name only. github_installation_id is
			// choice-validated server-side and 400s on a stale installation id, which
			// would turn "installation deleted out of band" into a plan failure
			// instead of a re-create; it is forbidden here so it is not re-added as
			// an optimisation. See FindGithubRepository.
			name: "github filters by repository name, never by installation id",
			row: map[string]any{
				"id": "gh-row", "github_repository_id": 42,
				"github_installation":   map[string]string{"id": "inst-gh"},
				"repository":            map[string]string{"name": "o/r", "url": "https://gh/o/r"},
				"repository_context_id": "ctx",
			},
			find: func(c *APIClient) (*ScmRepository, error) {
				return c.FindGithubRepository("inst-gh", "o/r", 42)
			},
			want:   url.Values{"search": {"o/r"}, "search_fields": {"repository_name"}},
			forbid: []string{"github_installation_id"},
		},
		{
			name: "gitlab filters by installation and project id",
			row: map[string]any{
				"id": "gl-row", "gitlab_project_id": 7,
				"gitlab_installation":   map[string]string{"id": "inst-gl"},
				"repository":            map[string]string{"name": "g/p", "url": "https://gl/g/p"},
				"repository_context_id": "ctx",
			},
			find: func(c *APIClient) (*ScmRepository, error) {
				return c.FindGitlabRepository("inst-gl", 7)
			},
			want: url.Values{"installation_id": {"inst-gl"}, "gitlab_project_id": {"7"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got url.Values
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.URL.Query()
				_ = json.NewEncoder(w).Encode(map[string]any{
					"total_items": 1,
					"data":        []map[string]any{tc.row},
				})
			}))
			defer srv.Close()

			client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
			repo, err := tc.find(client)
			if err != nil {
				t.Fatal(err)
			}
			if repo == nil {
				t.Fatal("expected to find the row")
			}
			for key, want := range tc.want {
				if got.Get(key) != want[0] {
					t.Errorf("filter %s = %q, want %q (full query: %v)", key, got.Get(key), want[0], got)
				}
			}
			for _, key := range tc.forbid {
				if got.Has(key) {
					t.Errorf("filter %s must not be sent, got %q", key, got.Get(key))
				}
			}
			// Pagination params must survive alongside the filters.
			if got.Get("limit") == "" || got.Get("start_at_index") == "" {
				t.Errorf("pagination params lost: %v", got)
			}
		})
	}
}

// GitHub's name filter narrows but does not identify, so a miss must never be
// read as "repository deleted" — that would plan a destroy/re-create of a live
// repository whose name drifted (renamed on GitHub, or absent right after an
// import). These cases pin the hint-plus-fallback shape and its request count.
func TestFindGithubRepository_NameFilterIsOnlyAHint(t *testing.T) {
	const row = `{"id":"gh-row","github_repository_id":42,
		"github_installation":{"id":"inst-gh"},
		"repository":{"name":"acme/repo","url":"https://gh/acme/repo"},
		"repository_context_id":"ctx"}`

	tests := []struct {
		name         string
		repoName     string
		wantRequests []string // the "search" value of each expected request, in order
	}{
		{
			// The hint resolves the row, so the unfiltered scan is never paid for.
			name:         "matching name resolves in one filtered request",
			repoName:     "acme/repo",
			wantRequests: []string{"acme/repo"},
		},
		{
			// A stale name narrows to nothing; the unfiltered retry still finds it.
			name:         "stale name falls back to the unfiltered scan",
			repoName:     "renamed-away/repo",
			wantRequests: []string{"renamed-away/repo", ""},
		},
		{
			// Post-import state carries no name; skip straight to the scan.
			name:         "unknown name skips the filtered query",
			repoName:     "",
			wantRequests: []string{""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var searches []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				search := r.URL.Query().Get("search")
				searches = append(searches, search)
				// Stand in for the search backend: the row matches only its own name.
				if search != "" && search != "acme/repo" {
					_, _ = w.Write([]byte(`{"total_items":0,"data":[]}`))
					return
				}
				_, _ = w.Write([]byte(`{"total_items":1,"data":[` + row + `]}`))
			}))
			defer srv.Close()

			client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
			found, err := client.FindGithubRepository("inst-gh", tc.repoName, 42)
			if err != nil {
				t.Fatal(err)
			}
			if found == nil || found.ID != "gh-row" {
				t.Fatalf("repository not found: %+v", found)
			}
			if len(searches) != len(tc.wantRequests) {
				t.Fatalf("made %d requests %q, want %d %q",
					len(searches), searches, len(tc.wantRequests), tc.wantRequests)
			}
			for i, want := range tc.wantRequests {
				if searches[i] != want {
					t.Errorf("request %d sent search=%q, want %q", i, searches[i], want)
				}
			}
		})
	}
}

// The fallback must not resurrect a repository that is genuinely gone: both the
// filtered and unfiltered lookups miss, and the caller still gets "not found" so
// Terraform can plan a replacement.
func TestFindGithubRepository_DeletedRepositoryStaysNotFound(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"total_items":0,"data":[]}`))
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	found, err := client.FindGithubRepository("inst-gh", "acme/repo", 42)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Fatalf("expected not found, got %+v", found)
	}
	if requests != 2 {
		t.Errorf("expected the filtered miss to retry unfiltered, got %d requests", requests)
	}
}

// A filter the API does not support is silently ignored, so the local match is
// still what guarantees we return the right row.
func TestFindRepository_LocalMatchGuardsIgnoredFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate an API that ignores every filter and returns two installations' rows.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_items": 2,
			"data": []map[string]any{
				{
					"id": "wrong", "gitlab_project_id": 7,
					"gitlab_installation":   map[string]string{"id": "other-installation"},
					"repository":            map[string]string{"name": "other/p"},
					"repository_context_id": "ctx-other",
				},
				{
					"id": "right", "gitlab_project_id": 7,
					"gitlab_installation":   map[string]string{"id": "inst-gl"},
					"repository":            map[string]string{"name": "mine/p"},
					"repository_context_id": "ctx-mine",
				},
			},
		})
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	repo, err := client.FindGitlabRepository("inst-gl", 7)
	if err != nil {
		t.Fatal(err)
	}
	if repo == nil {
		t.Fatal("expected a match")
	}
	if repo.ID != "right" {
		t.Errorf("matched the wrong installation's row: got id=%q, want %q", repo.ID, "right")
	}
}
