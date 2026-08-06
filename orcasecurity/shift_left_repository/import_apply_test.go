package shift_left_repository_test

// Import+apply against in-process stub: branch is create-only and never returned, so import must not RequireReplace on null→config.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

const (
	stubRepoAccountID    = "33333333-3333-3333-3333-333333333333"
	stubRepoRowID        = "44444444-4444-4444-4444-444444444444"
	stubRepoContextID    = "55555555-5555-5555-5555-555555555555"
	stubGithubRepoID     = "1125049043"
	stubRepoResourceName = "orcasecurity_shift_left_github_repository.test"
)

// repoStub never returns branch, matching the list API.
type repoStub struct {
	mu         sync.Mutex
	row        map[string]any
	patches    int
	integrates int
	deletes    int
}

func newRepoStub() *repoStub {
	return &repoStub{
		row: map[string]any{
			"id": stubRepoRowID,
			"repository": map[string]any{
				"name": "acme/service-api",
				"url":  "https://github.com/acme/service-api",
			},
			"status":                     "SUCCESS",
			"repository_context_id":      stubRepoContextID,
			"integration_status":         "ENABLED",
			"disabled":                   false,
			"disable_scan_pull_requests": false,
			"comments_on_pull_requests":  "ALWAYS",
			"pr_summary_comment":         "ALWAYS",
			"skip_check_runs":            "ALWAYS",
			"config_file_support":        "ENABLED",
			"github_repository_id":       1125049043,
			"github_installation":        map[string]any{"id": stubRepoAccountID},
		},
	}
}

func (s *repoStub) applyPatch(body map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.patches++
	for _, key := range []string{
		"comments_on_pull_requests", "pr_summary_comment", "skip_check_runs", "config_file_support",
	} {
		if value, ok := body[key].(string); ok && value != "" {
			s.row[key] = value
		}
	}
	if value, ok := body["disable_scan_pull_requests"].(bool); ok {
		s.row["disable_scan_pull_requests"] = value
	}
	if value, ok := body["disabled"].(bool); ok {
		s.row["disabled"] = value
	}
}

func (s *repoStub) snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]any, len(s.row))
	for key, value := range s.row {
		out[key] = value
	}
	return out
}

func (s *repoStub) counts() (patches, integrates, deletes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.patches, s.integrates, s.deletes
}

func (s *repoStub) start(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		isRepoPath := strings.Contains(r.URL.Path, "/github/integrated_repositories/")
		switch {
		case r.Method == http.MethodPost && isRepoPath:
			s.mu.Lock()
			s.integrates++
			s.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		case r.Method == http.MethodPatch && isRepoPath:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.applyPatch(body)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		case r.Method == http.MethodGet && isRepoPath:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data":        []map[string]any{s.snapshot()},
			})
		case r.Method == http.MethodDelete:
			s.mu.Lock()
			s.deletes++
			s.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"total_items": 0, "data": []map[string]any{}})
		}
	}))
	t.Cleanup(srv.Close)

	t.Setenv("TF_ACC", "1")
	t.Setenv("ORCASECURITY_API_ENDPOINT", srv.URL)
	t.Setenv("ORCASECURITY_API_TOKEN", "stub-token")
}

func stubRepoConfig() string {
	return orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_github_repository" "test" {
  account_id           = %q
  github_repository_id = %s
  name                 = "acme/service-api"
  url                  = "https://github.com/acme/service-api"
  branch               = "main"
}
`, stubRepoAccountID, stubGithubRepoID)
}

// Post-import apply must update in place, not DestroyBeforeCreate on null→config branch.
func TestGithubRepositoryImport_ThenApplyUpdatesInPlace(t *testing.T) {
	stub := newRepoStub()
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             stubRepoConfig(),
				ResourceName:       stubRepoResourceName,
				ImportState:        true,
				ImportStateId:      stubRepoAccountID + ":" + stubGithubRepoID,
				ImportStatePersist: true,
			},
			{
				Config: stubRepoConfig(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(stubRepoResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// The configured branch is recorded, since the API can never supply it.
					resource.TestCheckResourceAttr(stubRepoResourceName, "branch", "main"),
					resource.TestCheckResourceAttr(stubRepoResourceName, "account_id", stubRepoAccountID),
					resource.TestCheckResourceAttr(stubRepoResourceName, "github_repository_id", stubGithubRepoID),
					resource.TestCheckResourceAttr(stubRepoResourceName, "id", stubRepoRowID),
					resource.TestCheckResourceAttr(stubRepoResourceName, "repository_context_id", stubRepoContextID),
				),
			},
		},
	})

	// Only final-destroy DELETE expected — no re-integration POST from a replacement.
	_, integrates, deletes := stub.counts()
	if integrates != 0 {
		t.Errorf("import must not re-integrate the repository, got %d POSTs", integrates)
	}
	if deletes != 1 {
		t.Errorf("expected exactly the final-destroy DELETE, got %d", deletes)
	}
}

// A genuine branch change is still create-only and must force re-integration.
func TestGithubRepositoryImport_BranchChangeStillReplaces(t *testing.T) {
	stub := newRepoStub()
	stub.start(t)

	changedBranch := strings.Replace(stubRepoConfig(), `branch               = "main"`, `branch               = "develop"`, 1)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             stubRepoConfig(),
				ResourceName:       stubRepoResourceName,
				ImportState:        true,
				ImportStateId:      stubRepoAccountID + ":" + stubGithubRepoID,
				ImportStatePersist: true,
			},
			{
				Config: stubRepoConfig(),
				Check:  resource.TestCheckResourceAttr(stubRepoResourceName, "branch", "main"),
			},
			{
				Config: changedBranch,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(stubRepoResourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr(stubRepoResourceName, "branch", "develop"),
			},
		},
	})

	_, integrates, deletes := stub.counts()
	if integrates == 0 {
		t.Error("a branch change must re-integrate the repository")
	}
	// One DELETE for the replacement's destroy half, one for the end-of-test destroy.
	if deletes != 2 {
		t.Errorf("expected replace + final-destroy DELETEs, got %d", deletes)
	}
}
