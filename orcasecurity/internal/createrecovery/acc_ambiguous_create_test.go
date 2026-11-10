package createrecovery_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/backoff"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// accCase is a live create that the fault proxy turns ambiguous. post is the create path and method
// its verb (default POST). ids is the natural-key lister recovery uses; the committed mode's
// exactly-one-copy check is what proves it sees a new object, so the dropped mode's zero-copies check
// is not trivially true.
type accCase struct {
	name    string
	address string
	method  string
	post    string
	config  func(name string) string
	ids     func(c *api_client.APIClient, name string) (map[string]bool, error)
}

// accCases is filled by each domain's acc_*_test.go init.
var accCases []accCase

// faultProxy forwards everything to the lab, except the first create (method on post): with commit it
// is forwarded (the lab saves it) and the client gets a 504 anyway; without commit it is dropped.
type faultProxy struct {
	mu       sync.Mutex
	upstream *httputil.ReverseProxy
	method   string
	post     string
	commit   bool
	injected bool
	posts    int
}

func (p *faultProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == p.method && strings.TrimSuffix(r.URL.Path, "/") == strings.TrimSuffix(p.post, "/") {
		p.mu.Lock()
		p.posts++
		inject := !p.injected
		p.injected = true
		p.mu.Unlock()
		if inject {
			if p.commit {
				p.upstream.ServeHTTP(httptest.NewRecorder(), r)
			}
			http.Error(w, "<html>504 Gateway Time-out (injected)</html>", http.StatusGatewayTimeout)
			return
		}
	}
	p.upstream.ServeHTTP(w, r)
}

func (p *faultProxy) postCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.posts
}

func startFaultProxy(t *testing.T, c accCase, commit bool) (*faultProxy, *api_client.APIClient) {
	t.Helper()
	endpoint, token := os.Getenv("ORCASECURITY_API_ENDPOINT"), os.Getenv("ORCASECURITY_API_TOKEN")
	if os.Getenv("TF_ACC") == "" || endpoint == "" || token == "" {
		t.Skip("TF_ACC, ORCASECURITY_API_ENDPOINT and ORCASECURITY_API_TOKEN must be set")
	}
	target, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	upstream := httputil.NewSingleHostReverseProxy(target)
	director := upstream.Director
	upstream.Director = func(r *http.Request) {
		director(r)
		r.Host = target.Host
	}
	method := c.method
	if method == "" {
		method = http.MethodPost
	}
	proxy := &faultProxy{upstream: upstream, method: method, post: c.post, commit: commit}
	srv := httptest.NewServer(proxy)
	t.Cleanup(srv.Close)
	direct, _ := api_client.NewAPIClient(&endpoint, &token)
	t.Setenv("ORCASECURITY_API_ENDPOINT", srv.URL)
	return proxy, direct
}

func accName(c accCase) string {
	return fmt.Sprintf("tf-acc-wasp1876-%s-%d", strings.ReplaceAll(c.name, "_", "-"), time.Now().UnixNano()%1_000_000)
}

func labCopies(c accCase, direct *api_client.APIClient, name string, want int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		ids, err := c.ids(direct, name)
		if err != nil {
			return fmt.Errorf("listing %s on the lab: %w", name, err)
		}
		if len(ids) != want {
			return fmt.Errorf("want %d %s named %q on the lab, found %d: %v", want, c.name, name, len(ids), ids)
		}
		return nil
	}
}

func onePost(proxy *faultProxy) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if got := proxy.postCount(); got != 1 {
			return fmt.Errorf("the ambiguous create must be sent once, the lab got %d %ss", got, proxy.method)
		}
		return nil
	}
}

// TestAccAmbiguousCreate_CommittedPostIsAdopted: the lab saves the create but the provider sees a
// 504. The object must be adopted (one POST, one copy on the lab, clean follow-up plan).
func TestAccAmbiguousCreate_CommittedPostIsAdopted(t *testing.T) {
	for _, c := range accCases {
		t.Run(c.name, func(t *testing.T) {
			proxy, direct := startFaultProxy(t, c, true)
			name := accName(c)
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config: orcasecurity.TestProviderConfig + c.config(name),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet(c.address, "id"),
						onePost(proxy),
						labCopies(c, direct, name, 1),
					),
				}},
				CheckDestroy: labCopies(c, direct, name, 0),
			})
		})
	}
}

// TestAccAmbiguousCreate_DroppedPostSaysCheckUI: the create never reaches the lab. The apply must
// fail with the check-the-Orca-UI error, without re-sending, and leave nothing behind.
func TestAccAmbiguousCreate_DroppedPostSaysCheckUI(t *testing.T) {
	for _, c := range accCases {
		t.Run(c.name, func(t *testing.T) {
			backoff.FastForTest(t)
			proxy, direct := startFaultProxy(t, c, false)
			name := accName(c)
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      orcasecurity.TestProviderConfig + c.config(name),
					ExpectError: regexp.MustCompile(`check\s+the\s+Orca\s+UI`),
				}},
				CheckDestroy: resource.ComposeAggregateTestCheckFunc(onePost(proxy), labCopies(c, direct, name, 0)),
			})
		})
	}
}
