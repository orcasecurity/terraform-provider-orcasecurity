package shift_left_unit_test

// Plan-time coverage for the integrate guard: a create that can only be a
// fresh integrate must fail during plan, while the same configuration adopts
// an existing account without complaint. Uses the same in-process stub as
// create_apply_test.go, so it runs in normal CI.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// startAbsentUnitStub serves an empty unit list: every lookup misses, so a
// create can only be a fresh integrate.
func startAbsentUnitStub(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"total_items": 0, "data": []map[string]any{}})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TF_ACC", "1")
	t.Setenv("ORCASECURITY_API_ENDPOINT", srv.URL)
	t.Setenv("ORCASECURITY_API_TOKEN", "stub-token")
}

// A fresh integrate cannot use SELECTED_REPOSITORIES (the API wants an
// explicit repository list this resource never sends). The plan, not the
// apply, must say so.
func TestScmUnitPlan_FreshIntegrateRejectsSelectedRepositories(t *testing.T) {
	startAbsentUnitStub(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: stubConfig(`
  installation_mode = "SELECTED_REPOSITORIES"`),
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`SCAN_ALL_INCLUDE_FUTURE`),
		}},
	})
}

// The same mode is legal when the unit already exists: create adopts it via
// the update endpoint, which accepts SELECTED_REPOSITORIES.
func TestScmUnitPlan_AdoptAllowsSelectedRepositories(t *testing.T) {
	stub := newSCMUnitStub()
	stub.start(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: stubConfig(`
  installation_mode = "SELECTED_REPOSITORIES"`),
			Check: resource.TestCheckResourceAttr(stubResourceName, "installation_mode", "SELECTED_REPOSITORIES"),
		}},
	})
}
