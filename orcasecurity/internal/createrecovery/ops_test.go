package createrecovery_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/backoff"
	"terraform-provider-orcasecurity/orcasecurity/internal/createrecovery"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

const (
	newID   = "11111111-1111-1111-1111-111111111111"
	oldID   = "22222222-2222-2222-2222-222222222222"
	stubTag = "tf-recovery"
)

type runner func(context.Context, *diag.Diagnostics) (string, error)

// opCase is one create under test. path is the create path and method its verb (default POST); get
// answers every GET by path (query stripped) with created reporting whether the ambiguous create
// committed. A preexisting same-named object (oldID) must always be listed so the test proves it is
// never adopted. empty is the page after the last one for offset-paginated lists (default {"data":[]}).
// answer, when set, is a 200 body the create gets instead of a 504.
type opCase struct {
	name    string
	method  string
	path    string
	get     map[string]func(created bool) string
	empty   string
	adopted string
	answer  string
	op      func(c *api_client.APIClient) runner
}

func createMethod(method string) string {
	if method == "" {
		return http.MethodPost
	}
	return method
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
	creates int
}

func (s *ambiguousStub) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.Method == createMethod(s.c.method) && r.URL.Path == s.c.path {
		s.creates++
		s.created = s.commits
		if s.c.answer != "" {
			_, _ = w.Write([]byte(s.c.answer))
			return
		}
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
	backoff.FastForTest(t)
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

func TestCreateOps_AmbiguousCreateThatCommittedIsAdopted(t *testing.T) {
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
			if stub.creates != 1 {
				t.Errorf("an ambiguous create must not be replayed, got %d creates", stub.creates)
			}
		})
	}
}

func TestCreateOps_AmbiguousCreateThatDidNotCommitSaysCheckUI(t *testing.T) {
	for _, c := range opCases {
		t.Run(c.name, func(t *testing.T) {
			stub, client := startStub(t, c, false)
			var diags diag.Diagnostics
			_, err := c.op(client)(context.Background(), &diags)
			if err == nil || !strings.Contains(err.Error(), "check the Orca UI") {
				t.Fatalf("err = %v, want a check-the-Orca-UI error", err)
			}
			if stub.creates != 1 {
				t.Errorf("an ambiguous create must not be replayed, got %d creates", stub.creates)
			}
		})
	}
}
