package shift_left_policy_test

// Creating a policy that attaches projects takes two writes: POST the policy, then PUT its projects.
// Terraform records no state when Create reports an error, so a policy that survives a failed attach
// is invisible to Terraform and the next apply creates a second one. These tests drive the whole
// resource against an in-process stub of the Orca policy API, so they need no credentials and run in
// normal CI. `malicious_packages` is used because it has no controls and no catalog, which keeps the
// stub to the four endpoints the create path actually walks.

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
	// readBackMissing makes the post-attach GET 404, standing in for a policy the API
	// accepted but cannot serve back yet.
	readBackMissing bool
	requests        []string
	projects        []string
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

func (s *policyStub) start(t *testing.T) {
	t.Helper()
	const base = "/api/shiftleft/malicious_packages/policies/"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.record(r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(s.policyBody())
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/projects/"):
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
		case r.Method == http.MethodGet && r.URL.Path == base+stubPolicyID+"/":
			if s.readBackMissing {
				http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(s.policyBody())
		case r.Method == http.MethodDelete:
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

// A policy that cannot be read back is equally untracked, so it has to be rolled back too rather
// than left behind for the next apply to duplicate.
func TestShiftLeftPolicyCreate_FailedReadBackDeletesPolicy(t *testing.T) {
	stub := &policyStub{attachStatus: http.StatusOK, readBackMissing: true}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      stubPolicyConfig,
			ExpectError: regexp.MustCompile(`read back after attaching projects`),
		}},
	})

	if got := stub.deletes(); got != 1 {
		t.Errorf("a policy that cannot be read back must be deleted, got %d DELETEs in %v",
			got, stub.recorded())
	}
}

// A create that succeeds end to end must not delete anything. resource.Test also refreshes and
// re-plans after the apply, so this step doubles as the check that a policy created without a
// description settles instead of replanning an in-place update forever.
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

	// resource.Test destroys at the end of the case, so exactly the teardown DELETE is expected.
	if got := stub.deletes(); got != 1 {
		t.Errorf("expected only the final destroy DELETE, got %d in %v", got, stub.recorded())
	}
}
