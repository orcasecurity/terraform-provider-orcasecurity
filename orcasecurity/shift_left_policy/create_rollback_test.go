package shift_left_policy_test

// Create is POST then projects PUT; failed attach leaves an untracked policy unless rolled back. Stub uses malicious_packages (no controls/catalog).

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

const stubPolicyID = "77777777-7777-7777-7777-777777777777"

type policyStub struct {
	mu sync.Mutex
	// attachStatus is the status the projects PUT answers with.
	attachStatus int
	// readBackMissing: post-attach GET 404 — API accepted the policy but cannot serve it yet.
	readBackMissing bool
	requests        []string
	projects        []string
	// postStatus, when set, answers the create POST with this status; postCommits says whether the
	// policy was created anyway (an ambiguous gateway/timeout failure).
	postStatus  int
	postCommits bool
	created     bool
	// preexisting are same-name policies that existed before this create.
	preexisting []string
}

func (s *policyStub) record(method, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, method+" "+path)
}

func (s *policyStub) attach(projects []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects = projects
}

func (s *policyStub) policyBody() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	projects := s.projects
	if projects == nil {
		projects = []string{}
	}
	return map[string]any{
		"id": stubPolicyID, "name": "tf-stub-policy", "type": "malicious_packages",
		"is_builtin": false, "disabled": false, "warn_mode": false,
		"priority_failure_threshold": "HIGH", "projects_ids": projects,
	}
}

func (s *policyStub) listPage(startAt string) []map[string]any {
	if startAt != "" && startAt != "0" {
		return []map[string]any{}
	}
	s.mu.Lock()
	ids := append([]string(nil), s.preexisting...)
	if s.created {
		ids = append(ids, stubPolicyID)
	}
	s.mu.Unlock()
	rows := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, map[string]any{"id": id, "name": "tf-stub-policy", "type": "malicious_packages", "builtin": false})
	}
	return rows
}

func (s *policyStub) posts() int {
	count := 0
	for _, req := range s.recorded() {
		if strings.HasPrefix(req, "POST ") {
			count++
		}
	}
	return count
}

func (s *policyStub) recorded() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func (s *policyStub) deletes() int {
	count := 0
	for _, req := range s.recorded() {
		if strings.HasPrefix(req, "DELETE ") {
			count++
		}
	}
	return count
}

const stubPolicyBase = "/api/shiftleft/malicious_packages/policies/"

func (s *policyStub) start(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(srv.Close)

	t.Setenv("TF_ACC", "1")
	t.Setenv("ORCASECURITY_API_ENDPOINT", srv.URL)
	t.Setenv("ORCASECURITY_API_TOKEN", "stub-token")
}

func (s *policyStub) serve(w http.ResponseWriter, r *http.Request) {
	s.record(r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/shiftleft/policies/":
		_ = json.NewEncoder(w).Encode(map[string]any{"data": s.listPage(r.URL.Query().Get("start_at_index"))})
	case r.Method == http.MethodPost && r.URL.Path == stubPolicyBase:
		s.serveCreate(w)
	case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/projects/"):
		s.serveAttach(w, r)
	case r.Method == http.MethodGet && r.URL.Path == stubPolicyBase+stubPolicyID+"/":
		s.serveGet(w)
	case r.Method == http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "unexpected request "+r.Method+" "+r.URL.Path, http.StatusInternalServerError)
	}
}

func (s *policyStub) serveCreate(w http.ResponseWriter) {
	s.mu.Lock()
	s.created = s.postStatus == 0 || s.postCommits
	status := s.postStatus
	s.mu.Unlock()
	if status != 0 {
		http.Error(w, `<html>gateway timeout</html>`, status)
		return
	}
	_ = json.NewEncoder(w).Encode(s.policyBody())
}

func (s *policyStub) serveAttach(w http.ResponseWriter, r *http.Request) {
	if s.attachStatus != http.StatusOK {
		http.Error(w, `{"detail":"project attach rejected"}`, s.attachStatus)
		return
	}
	var body struct {
		ProjectsIds []string `json:"projects_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.attach(body.ProjectsIds)
	_ = json.NewEncoder(w).Encode(s.policyBody())
}

func (s *policyStub) serveGet(w http.ResponseWriter) {
	if s.readBackMissing {
		http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(s.policyBody())
}

const stubPolicyConfig = orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_policy" "stub" {
  type                       = "malicious_packages"
  name                       = "tf-stub-policy"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"
  projects_ids               = ["proj-1"]
}
`

func TestShiftLeftPolicyCreate_FailedProjectAttachDeletesPolicy(t *testing.T) {
	stub := &policyStub{attachStatus: http.StatusInternalServerError}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      stubPolicyConfig,
			ExpectError: regexp.MustCompile(`Error setting AppSec policy projects`),
		}},
	})

	if got := stub.deletes(); got != 1 {
		t.Errorf("the created policy must be deleted when the project attach fails, got %d DELETEs in %v",
			got, stub.recorded())
	}
}

// Unreadable after attach is untracked too — roll back or the next apply duplicates.
func TestShiftLeftPolicyCreate_FailedReadBackDeletesPolicy(t *testing.T) {
	stub := &policyStub{attachStatus: http.StatusOK, readBackMissing: true}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      stubPolicyConfig,
			ExpectError: regexp.MustCompile(`read back after create`),
		}},
	})

	if got := stub.deletes(); got != 1 {
		t.Errorf("a policy that cannot be read back must be deleted, got %d DELETEs in %v",
			got, stub.recorded())
	}
}

// Success must not DELETE; refresh/re-plan also checks a description-less create settles.
func TestShiftLeftPolicyCreate_SuccessKeepsPolicy(t *testing.T) {
	stub := &policyStub{attachStatus: http.StatusOK}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: stubPolicyConfig,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.stub", "id", stubPolicyID),
				resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.stub", "projects_ids.0", "proj-1"),
			),
		}},
	})

	// Only resource.Test teardown DELETE is expected.
	if got := stub.deletes(); got != 1 {
		t.Errorf("expected only the final destroy DELETE, got %d in %v", got, stub.recorded())
	}
}

// A 504 on create can still have committed the policy. The client must not replay the POST, and
// Create must adopt the new policy instead of leaving it untracked (seen live on the lab).
func TestShiftLeftPolicyCreate_AmbiguousPostThatCommittedIsAdopted(t *testing.T) {
	stub := &policyStub{attachStatus: http.StatusOK, postStatus: http.StatusGatewayTimeout, postCommits: true,
		preexisting: []string{"11111111-1111-1111-1111-111111111111"}}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: stubPolicyConfig,
			Check:  resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.stub", "id", stubPolicyID),
		}},
	})

	if got := stub.posts(); got != 1 {
		t.Errorf("an ambiguous create must not be replayed, got %d POSTs in %v", got, stub.recorded())
	}
}

func TestShiftLeftPolicyCreate_AmbiguousPostThatDidNotCommitSaysSafeToRetry(t *testing.T) {
	stub := &policyStub{attachStatus: http.StatusOK, postStatus: http.StatusGatewayTimeout,
		preexisting: []string{"11111111-1111-1111-1111-111111111111"}}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      stubPolicyConfig,
			ExpectError: regexp.MustCompile(`was\s+not\s+created\s+and\s+re-applying\s+is\s+safe`),
		}},
	})

	if got := stub.posts(); got != 1 {
		t.Errorf("an ambiguous create must not be replayed, got %d POSTs in %v", got, stub.recorded())
	}
}
