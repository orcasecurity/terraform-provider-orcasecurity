package createrecovery_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/createrecovery"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

const (
	newID   = "11111111-1111-1111-1111-111111111111"
	oldID   = "22222222-2222-2222-2222-222222222222"
	stubTag = "tf-recovery"
)

type runner func(context.Context, *diag.Diagnostics) (string, error)

// opCase is one create under test. post is the create path; get answers every GET by path (query
// stripped) with created reporting whether the ambiguous POST committed. A preexisting same-named
// object (oldID) must always be listed so the test proves it is never adopted. empty is the page
// after the last one for offset-paginated lists (default {"data":[]}).
type opCase struct {
	name    string
	post    string
	get     map[string]func(created bool) string
	empty   string
	adopted string
	op      func(c *api_client.APIClient) runner
}

// opCases is filled by each domain's *_ops_test.go init.
var opCases []opCase

func runOp[T any](op api_client.CreateOp[T]) runner {
	return func(ctx context.Context, diags *diag.Diagnostics) (string, error) {
		got, err := createrecovery.Run(ctx, op, diags)
		if err != nil {
			return "", err
		}
		return op.ID(got), nil
	}
}

type ambiguousStub struct {
	mu      sync.Mutex
	c       opCase
	commits bool
	created bool
	posts   int
}

func (s *ambiguousStub) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost && r.URL.Path == s.c.post {
		s.posts++
		s.created = s.commits
		http.Error(w, "<html>gateway timeout</html>", http.StatusGatewayTimeout)
		return
	}
	body, ok := s.c.get[r.URL.Path]
	if r.Method != http.MethodGet || !ok {
		http.Error(w, "unexpected request "+r.Method+" "+r.URL.String(), http.StatusInternalServerError)
		return
	}
	if start := r.URL.Query().Get("start_at_index"); start != "" && start != "0" {
		empty := s.c.empty
		if empty == "" {
			empty = `{"data":[]}`
		}
		_, _ = w.Write([]byte(empty))
		return
	}
	_, _ = w.Write([]byte(body(s.created)))
}

func startStub(t *testing.T, c opCase, commits bool) (*ambiguousStub, *api_client.APIClient) {
	t.Helper()
	createrecovery.FastForTest(t)
	stub := &ambiguousStub{c: c, commits: commits}
	srv := httptest.NewServer(http.HandlerFunc(stub.serve))
	t.Cleanup(srv.Close)
	endpoint, token := srv.URL, "stub-token"
	client, _ := api_client.NewAPIClient(&endpoint, &token)
	return stub, client
}

// rows renders the list body: the preexisting same-named object, plus the new one once committed.
func rows(created bool, row func(id string) string, wrap func(items string) string) string {
	items := row(oldID)
	if created {
		items += "," + row(newID)
	}
	return wrap(items)
}

func dataList(items string) string { return `{"data":[` + items + `]}` }

func TestCreateOps_AmbiguousPostThatCommittedIsAdopted(t *testing.T) {
	for _, c := range opCases {
		t.Run(c.name, func(t *testing.T) {
			stub, client := startStub(t, c, true)
			var diags diag.Diagnostics
			want := newID
			if c.adopted != "" {
				want = c.adopted
			}
			id, err := c.op(client)(context.Background(), &diags)
			if err != nil || id != want {
				t.Fatalf("id=%q err=%v, want %s", id, err, want)
			}
			if diags.WarningsCount() != 1 {
				t.Errorf("want one recovery warning, got %v", diags)
			}
			if stub.posts != 1 {
				t.Errorf("an ambiguous create must not be replayed, got %d POSTs", stub.posts)
			}
		})
	}
}

func TestCreateOps_AmbiguousPostThatDidNotCommitSaysCheckUI(t *testing.T) {
	for _, c := range opCases {
		t.Run(c.name, func(t *testing.T) {
			stub, client := startStub(t, c, false)
			var diags diag.Diagnostics
			_, err := c.op(client)(context.Background(), &diags)
			if err == nil || !strings.Contains(err.Error(), "check the Orca UI") {
				t.Fatalf("err = %v, want a check-the-Orca-UI error", err)
			}
			if stub.posts != 1 {
				t.Errorf("an ambiguous create must not be replayed, got %d POSTs", stub.posts)
			}
		})
	}
}
