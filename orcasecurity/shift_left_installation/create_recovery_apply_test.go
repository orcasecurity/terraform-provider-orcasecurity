package shift_left_installation_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/internal/createrecovery"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	stubInstallationsPath = "/api/shiftleft/gitlab/installations/"
	stubInstallationID    = "33333333-3333-3333-3333-333333333333"
	stubInstallationName  = "tf-stub-gitlab"
)

// installationStub: GitLab installations list/create/delete. postStatus answers the create; postCommits
// says whether the installation was created anyway (Orca validates the token against a slow SCM).
type installationStub struct {
	mu          sync.Mutex
	postStatus  int
	postCommits bool
	created     bool
	posts       int
	preexisting []string
}

func (s *installationStub) rows() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := append([]string(nil), s.preexisting...)
	if s.created {
		ids = append(ids, stubInstallationID)
	}
	rows := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, map[string]any{"id": id, "name": stubInstallationName, "server_url": "https://gitlab.com",
			"read_only": false, "cloud_integration": true})
	}
	return rows
}

func (s *installationStub) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == stubInstallationsPath:
		if q := r.URL.Query().Get("start_at_index"); q != "" && q != "0" {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": s.rows()})
	case r.Method == http.MethodPost && r.URL.Path == stubInstallationsPath:
		s.mu.Lock()
		s.posts++
		s.created = s.postStatus == 0 || s.postCommits
		status := s.postStatus
		s.mu.Unlock()
		if status != 0 {
			http.Error(w, `<html>gateway timeout</html>`, status)
			return
		}
		_ = json.NewEncoder(w).Encode(s.rows()[len(s.rows())-1])
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, stubInstallationsPath):
		s.mu.Lock()
		s.created = false
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "unexpected request "+r.Method+" "+r.URL.Path, http.StatusInternalServerError)
	}
}

func (s *installationStub) start(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(srv.Close)
	t.Setenv("TF_ACC", "1")
	t.Setenv("ORCASECURITY_API_ENDPOINT", srv.URL)
	t.Setenv("ORCASECURITY_API_TOKEN", "stub-token")
}

func (s *installationStub) postCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.posts
}

const stubInstallationConfig = orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_gitlab_installation" "stub" {
  name         = "tf-stub-gitlab"
  access_token = "glpat-stub"
}
`

func TestGitlabInstallationCreate_AmbiguousPostThatCommittedIsAdopted(t *testing.T) {
	createrecovery.FastForTest(t)
	stub := &installationStub{postStatus: http.StatusGatewayTimeout, postCommits: true,
		preexisting: []string{"44444444-4444-4444-4444-444444444444"}}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: stubInstallationConfig,
			Check:  resource.TestCheckResourceAttr("orcasecurity_shift_left_gitlab_installation.stub", "id", stubInstallationID),
		}},
	})
	if got := stub.postCount(); got != 1 {
		t.Errorf("an ambiguous create must not be replayed, got %d POSTs", got)
	}
}

func TestGitlabInstallationCreate_AmbiguousPostThatDidNotCommitSaysCheckUI(t *testing.T) {
	createrecovery.FastForTest(t)
	stub := &installationStub{postStatus: http.StatusGatewayTimeout}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      stubInstallationConfig,
			ExpectError: regexp.MustCompile(`no\s+new\s+GitLab\s+installation\s+named\s+"tf-stub-gitlab"\s+appeared[\s\S]*check\s+the\s+Orca\s+UI`),
		}},
	})
}
