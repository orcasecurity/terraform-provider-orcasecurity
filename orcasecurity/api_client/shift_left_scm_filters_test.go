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
			client := scmListClient(t, func(w http.ResponseWriter, r *http.Request) {
				got = r.URL.Query()
				_ = json.NewEncoder(w).Encode(map[string]any{
					"total_items": 1,
					"data":        firstPageRows(r, []map[string]any{tc.row}),
				})
			})

			repo, err := tc.find(client)
			if err != nil {
				t.Fatal(err)
			}
			if repo == nil {
				t.Fatal("expected to find the row")
			}
			assertScmListQuery(t, got, tc.want, tc.forbid)
		})
	}
}

func scmListClient(t *testing.T, handler http.HandlerFunc) *APIClient {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
}

// Wanted filters present, forbidden absent, paging params kept.
func assertScmListQuery(t *testing.T, got, want url.Values, forbid []string) {
	t.Helper()

	for key, values := range want {
		if got.Get(key) != values[0] {
			t.Errorf("filter %s = %q, want %q (full query: %v)", key, got.Get(key), values[0], got)
		}
	}
	for _, key := range forbid {
		if got.Has(key) {
			t.Errorf("filter %s must not be sent, got %q", key, got.Get(key))
		}
	}
	if got.Get("limit") == "" || got.Get("start_at_index") == "" {
		t.Errorf("pagination params lost: %v", got)
	}
}

// Name filter is hint-only; stale name must fall back to unfiltered scan (id survives renames).
func TestFindGithubRepository_NameFilterIsOnlyAHint(t *testing.T) {
	const row = `{"id":"gh-row","github_repository_id":42,
		"github_installation":{"id":"inst-gh"},
		"repository":{"name":"acme/repo","url":"https://gh/acme/repo"},
		"repository_context_id":"ctx"}`

	tests := []struct {
		name         string
		repoName     string
		wantFound    bool
		wantRequests []string // the "search" value of each expected request, in order
	}{
		{
			// Hint hit: one filtered request, no fallback scan.
			name:         "matching name resolves in one filtered request",
			repoName:     "acme/repo",
			wantFound:    true,
			wantRequests: []string{"acme/repo"},
		},
		{
			// Stale name misses filter; unfiltered scan finds row by id.
			name:         "stale name falls back to the unfiltered scan and still resolves",
			repoName:     "renamed-away/repo",
			wantFound:    true,
			wantRequests: []string{"renamed-away/repo", ""},
		},
		{
			// Empty name skips hint filter; scan only.
			name:         "unknown name skips the filtered query",
			repoName:     "",
			wantFound:    true,
			wantRequests: []string{""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			backend := &searchingList{rowName: "acme/repo", row: row}
			client := scmListClient(t, backend.handle)

			found, err := client.FindGithubRepository("inst-gh", tc.repoName, 42)
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantFound {
				if found == nil || found.ID != "gh-row" {
					t.Fatalf("repository not found: %+v", found)
				}
			} else if found != nil {
				t.Fatalf("expected not found, got %+v", found)
			}
			assertSearchSequence(t, backend.searches, tc.wantRequests)
		})
	}
}

// Fake list: row matches only its own name; records search query values.
type searchingList struct {
	rowName  string
	row      string
	searches []string
}

func (s *searchingList) handle(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	// Record only the first page of each scan attempt: the trailing empty-page
	// request that confirms termination is pagination plumbing, not a new attempt.
	if start := r.URL.Query().Get("start_at_index"); start == "" || start == "0" {
		s.searches = append(s.searches, search)
	}
	body := `{"total_items":0,"data":[]}`
	if search == "" || search == s.rowName {
		body = `{"total_items":1,"data":[` + s.row + `]}`
	}
	_, _ = w.Write([]byte(firstPageOnly(r, body)))
}

// Pin search values in order (request count separates hint-hit vs fallback).
func assertSearchSequence(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("made %d requests %q, want %d %q", len(got), got, len(want), want)
	}
	for i, search := range want {
		if got[i] != search {
			t.Errorf("request %d sent search=%q, want %q", i, got[i], search)
		}
	}
}

// Truly gone repo stays not found after filtered + unfiltered scan (rename vs delete).
func TestFindGithubRepository_DeletedRepositoryStaysNotFound(t *testing.T) {
	var requests int
	client := scmListClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"total_items":0,"data":[]}`))
	})

	found, err := client.FindGithubRepository("inst-gh", "acme/repo", 42)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Fatalf("expected not found, got %+v", found)
	}
	if requests != 2 {
		t.Errorf("expected the filtered request plus the unfiltered fallback, got %d requests", requests)
	}
}

// A filter the API does not support is silently ignored, so the local match is
// still what guarantees we return the right row.
func TestFindRepository_LocalMatchGuardsIgnoredFilters(t *testing.T) {
	client := scmListClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Simulate an API that ignores every filter and returns two installations' rows.
		rows := []map[string]any{
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
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"total_items": 2, "data": firstPageRows(r, rows)})
	})

	repo, err := client.FindGitlabRepository("inst-gl", 7)
	if err != nil {
		t.Fatal(err)
	}
	if repo == nil {
		t.Fatal("expected a match")
		return
	}
	if repo.ID != "right" {
		t.Errorf("matched the wrong installation's row: got id=%q, want %q", repo.ID, "right")
	}
}
