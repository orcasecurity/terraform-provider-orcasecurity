package shift_left_policy_test

// The API's last-active guard runs on every DELETE, so both paths that remove a policy — the user's
// destroy and Create's rollback after a failed apply — have to recover from it the same way. Driven
// through resource.Test so the recovery is exercised where it runs. Stub uses malicious_packages
// (no catalog). The direct-Delete cases live in delete_last_active_test.go.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// lastActivePolicyStub refuses a DELETE the way the API does while the policy is still attached,
// and optionally refuses the detach that would lift the refusal too.
type lastActivePolicyStub struct {
	mu sync.Mutex
	// allowDetach mirrors the API letting a detach through where it rejects the delete.
	allowDetach bool
	// readBackMissing 404s the post-attach GET, so Create rolls the new policy back.
	readBackMissing bool
	detached        bool
	detaches        int
	deletes         int
}

func (s *lastActivePolicyStub) counts() (detaches, deletes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.detaches, s.deletes
}

const lastActivePolicyError = `{"message":"Policy cannot be removed/disabled since it's the last active policy of the following projects: ['proj-1(11111111-1111-1111-1111-111111111111)']"}`

func (s *lastActivePolicyStub) body() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	projects := []map[string]any{{"id": "proj-1"}}
	if s.detached {
		projects = nil
	}
	return map[string]any{
		"id": stubPolicyID, "name": "tf-last-active", "type": "malicious_packages",
		"builtin": false, "disabled": false, "warn_mode": false,
		"priority_failure_threshold": "HIGH", "projects": projects,
	}
}

func (s *lastActivePolicyStub) handleProjectsPut(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ProjectsIds *[]string `json:"projects_ids"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	if payload.ProjectsIds != nil && len(*payload.ProjectsIds) == 0 {
		s.mu.Lock()
		s.detaches++
		allowed := s.allowDetach
		s.detached = allowed
		s.mu.Unlock()
		if !allowed {
			http.Error(w, lastActivePolicyError, http.StatusBadRequest)
			return
		}
	}
	_ = json.NewEncoder(w).Encode(s.body())
}

func (s *lastActivePolicyStub) start(t *testing.T) {
	t.Helper()
	const base = "/api/shiftleft/malicious_packages/policies/"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(s.body())
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/projects/"):
			s.handleProjectsPut(w, r)
		case r.Method == http.MethodGet && r.URL.Path == base+stubPolicyID+"/":
			if s.readBackMissing {
				http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(s.body())
		case r.Method == http.MethodDelete:
			s.mu.Lock()
			s.deletes++
			detached := s.detached
			s.mu.Unlock()
			if !detached {
				http.Error(w, lastActivePolicyError, http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request "+r.Method+" "+r.URL.Path, http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	t.Setenv("TF_ACC", "1")
	t.Setenv("ORCASECURITY_API_ENDPOINT", srv.URL)
	t.Setenv("ORCASECURITY_API_TOKEN", "stub-token")
}

const lastActivePolicyConfig = orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_policy" "last_active" {
  type                       = "malicious_packages"
  name                       = "tf-last-active"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"
  projects_ids               = ["proj-1"]
}
`

// The API's delete guard flags a project whose only *active* policy is a different policy, so a
// policy can be undeletable until detached. Destroy must recover via the detach the API does allow,
// instead of leaving the resource stuck in state.
func TestShiftLeftPolicyDelete_LastActivePolicyRetriesAfterDetach(t *testing.T) {
	stub := &lastActivePolicyStub{allowDetach: true}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: lastActivePolicyConfig,
		}},
	})

	detaches, deletes := stub.counts()
	if detaches != 1 {
		t.Errorf("expected one detach before the delete retry, got %d", detaches)
	}
	if deletes != 2 {
		t.Errorf("expected the delete to be retried after the detach, got %d DELETEs", deletes)
	}
}

// Create's rollback deletes a policy the API has already attached to projects, so it hits the same
// guard as a destroy. Without the recovery the policy survives untracked and the next apply
// duplicates it.
func TestShiftLeftPolicyCreate_RollbackRecoversFromLastActivePolicy(t *testing.T) {
	stub := &lastActivePolicyStub{allowDetach: true, readBackMissing: true}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      lastActivePolicyConfig,
			ExpectError: regexp.MustCompile(`read back after create`),
		}},
	})

	detaches, deletes := stub.counts()
	if detaches != 1 {
		t.Errorf("expected the rollback to detach before retrying the delete, got %d detaches", detaches)
	}
	if deletes != 2 {
		t.Errorf("expected the rollback delete to be retried after the detach, got %d DELETEs", deletes)
	}
}
