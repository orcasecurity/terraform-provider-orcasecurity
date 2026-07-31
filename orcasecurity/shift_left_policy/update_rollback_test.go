package shift_left_policy_test

// Update is two writes (policy body, then projects). When the projects PUT fails the body has
// already landed; Terraform discards the plan, so without compensation live ≠ state. This stub
// drives Create then a failing Update and asserts the prior body is restored.

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

type updateStub struct {
	mu           sync.Mutex
	requests     []string
	name         string
	projects     []string
	projectPuts  int
	failProjects bool // fail projects PUTs after the create attach (put #2+)
	// failReadWhileRenamed fails GETs while the updated body is live, standing in for a
	// policy the API accepted but cannot serve back. Once the restore PUT lands the old
	// name, reads recover — which is exactly what lets the assertion see the restored body.
	failReadWhileRenamed bool
}

func (s *updateStub) record(method, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, method+" "+path)
}

func (s *updateStub) recorded() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func (s *updateStub) body() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	projects := s.projects
	if projects == nil {
		projects = []string{}
	}
	return map[string]any{
		"id": stubPolicyID, "name": s.name, "type": "malicious_packages",
		"is_builtin": false, "disabled": false, "warn_mode": false,
		"priority_failure_threshold": "HIGH", "projects_ids": projects,
	}
}

func (s *updateStub) start(t *testing.T) {
	t.Helper()
	s.name = "tf-stub-policy"
	s.projects = []string{"proj-1"}
	const base = "/api/shiftleft/malicious_packages/policies/"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.record(r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(s.body())
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/projects/"):
			s.mu.Lock()
			s.projectPuts++
			fail := s.failProjects && s.projectPuts > 1
			s.mu.Unlock()
			if fail {
				http.Error(w, `{"detail":"project attach rejected"}`, http.StatusInternalServerError)
				return
			}
			var body struct {
				ProjectsIds []string `json:"projects_ids"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			s.mu.Lock()
			s.projects = body.ProjectsIds
			s.mu.Unlock()
			_ = json.NewEncoder(w).Encode(s.body())
		case r.Method == http.MethodPut && r.URL.Path == base+stubPolicyID+"/":
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			s.mu.Lock()
			if body.Name != "" {
				s.name = body.Name
			}
			s.mu.Unlock()
			_ = json.NewEncoder(w).Encode(s.body())
		case r.Method == http.MethodGet && r.URL.Path == base+stubPolicyID+"/":
			s.mu.Lock()
			failRead := s.failReadWhileRenamed && s.name == "tf-stub-policy-renamed"
			s.mu.Unlock()
			if failRead {
				http.Error(w, `{"detail":"transient"}`, http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(s.body())
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

func TestShiftLeftPolicyUpdate_FailedProjectAttachRestoresBody(t *testing.T) {
	stub := &updateStub{failProjects: true}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: stubPolicyConfig},
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_policy" "stub" {
  type                       = "malicious_packages"
  name                       = "tf-stub-policy-renamed"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"
  projects_ids               = ["proj-1", "proj-2"]
}
`,
				ExpectError: regexp.MustCompile(`Error updating AppSec policy projects`),
			},
		},
	})

	stub.mu.Lock()
	name := stub.name
	stub.mu.Unlock()
	if name != "tf-stub-policy" {
		t.Fatalf("expected prior body name restored after failed projects PUT, got %q; requests=%v",
			name, stub.recorded())
	}
}

// When the read-back after an update fails, both writes have already landed (body and
// projects), so the restore has to rewind both: the prior body and the prior attachments.
func TestShiftLeftPolicyUpdate_FailedReadBackRestoresBodyAndProjects(t *testing.T) {
	stub := &updateStub{failReadWhileRenamed: true}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: stubPolicyConfig},
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_policy" "stub" {
  type                       = "malicious_packages"
  name                       = "tf-stub-policy-renamed"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"
  projects_ids               = ["proj-1", "proj-2"]
}
`,
				ExpectError: regexp.MustCompile(`Error reading AppSec policy after update`),
			},
		},
	})

	stub.mu.Lock()
	name := stub.name
	projects := append([]string(nil), stub.projects...)
	stub.mu.Unlock()
	if name != "tf-stub-policy" {
		t.Fatalf("expected prior body restored after failed read-back, got %q; requests=%v",
			name, stub.recorded())
	}
	if len(projects) != 1 || projects[0] != "proj-1" {
		t.Fatalf("expected prior projects [proj-1] restored after failed read-back, got %v; requests=%v",
			projects, stub.recorded())
	}
}
