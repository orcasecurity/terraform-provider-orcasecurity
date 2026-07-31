package shift_left_bitbucket_account_test

// These tests drive a full terraform apply against a stateful in-process stub of the Orca
// SCM unit API. They need no credentials and no lab tenant, so they run in normal CI.
//
// They cover the shared shift_left_integration.AdoptedUnitOps create/update path, which
// orcasecurity_shift_left_github_account, _gitlab_group, _azure_devops_account and
// _bitbucket_account all use, plus the shared SharedScmConfigAttributes schema. Bitbucket
// is the stand-in because it has the simplest identity (installation id + account slug).
//
// resource.Test asserts the plan is empty after each apply, so every case here is also a
// guard against a value that applies successfully but never settles.

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
)

const (
	stubInstallationID = "11111111-1111-1111-1111-111111111111"
	stubOrcaAccountID  = "22222222-2222-2222-2222-222222222222"
	stubAccountSlug    = "acme-workspace"
	stubResourceName   = "orcasecurity_shift_left_bitbucket_account.test"
)

// scmUnitStub models the unit the way the API does: a PUT replaces configuration_settings
// wholesale and rebinds or clears the project, and every later read returns what was stored.
type scmUnitStub struct {
	mu   sync.Mutex
	unit map[string]any
	puts []map[string]any

	// volatileSideEffects makes a write move server-owned Computed fields the way
	// the real API does (scan-all enrols repositories; a project rebind re-evaluates
	// posture). Opt-in so the existing cases keep their fixed, side-effect-free unit.
	volatileSideEffects bool
}

func newSCMUnitStub() *scmUnitStub {
	return &scmUnitStub{
		unit: map[string]any{
			"id":                stubOrcaAccountID,
			"installation_id":   stubInstallationID,
			"account_id":        stubAccountSlug,
			"account_name":      "Acme Workspace",
			"installation_mode": "SELECTED_REPOSITORIES",
			"default_policies":  true,
			"configuration_settings": map[string]any{
				"disable_scan_pull_requests": false,
				"comments_on_pull_requests":  "ALWAYS",
				"pr_summary_comment":         "ALWAYS",
				"skip_check_runs":            "ALWAYS",
				"config_file_support":        "ENABLED",
				"pr_summary_appendix":        "",
			},
		},
	}
}

func (s *scmUnitStub) applyPut(body map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.puts = append(s.puts, body)

	if cfg, ok := body["configuration_settings"].(map[string]any); ok {
		s.unit["configuration_settings"] = cfg
	}
	if mode, ok := body["installation_mode"].(string); ok && mode != "" {
		s.unit["installation_mode"] = mode
		if s.volatileSideEffects && mode == "SCAN_ALL_INCLUDE_FUTURE" {
			s.unit["integrated_repositories_count"] = 37
			s.unit["scan_all_state"] = "RUNNING"
		}
	}
	if defaultPolicies, ok := body["default_policies"].(bool); ok {
		s.unit["default_policies"] = defaultPolicies
	}
	// project_id is omitempty on the wire, so an absent or empty key means "unbound".
	if projectID, ok := body["project_id"].(string); ok && projectID != "" {
		s.unit["project"] = map[string]any{"id": projectID}
		if s.volatileSideEffects {
			s.unit["scm_posture_policy_id"] = "policy-" + projectID
		}
	} else {
		delete(s.unit, "project")
	}
}

func (s *scmUnitStub) snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]any, len(s.unit))
	for key, value := range s.unit {
		out[key] = value
	}
	return out
}

// lastPutConfigSettings returns the configuration_settings of the most recent write.
func (s *scmUnitStub) lastPutConfigSettings(t *testing.T) map[string]any {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.puts) == 0 {
		t.Fatal("no PUT was issued")
	}
	cfg, ok := s.puts[len(s.puts)-1]["configuration_settings"].(map[string]any)
	if !ok {
		t.Fatalf("last PUT carried no configuration_settings: %+v", s.puts[len(s.puts)-1])
	}
	return cfg
}

// start serves the unit list and config PUT, and points the provider at itself.
func (s *scmUnitStub) start(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		isUnitPath := strings.Contains(r.URL.Path, "/integrated_accounts/")
		switch {
		case r.Method == http.MethodPut && isUnitPath:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.applyPut(body)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		case r.Method == http.MethodGet && isUnitPath:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data":        []map[string]any{s.snapshot()},
			})
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"total_items": 0, "data": []map[string]any{}})
		}
	}))
	t.Cleanup(srv.Close)

	// The stub replaces the real API, so these tests are self-contained: TF_ACC only gates
	// resource.Test, and the token is never validated by the stub.
	t.Setenv("TF_ACC", "1")
	t.Setenv("ORCASECURITY_API_ENDPOINT", srv.URL)
	t.Setenv("ORCASECURITY_API_TOKEN", "stub-token")
}

func stubConfig(body string) string {
	return orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_bitbucket_account" "test" {
  installation_id = %q
  account_id      = %q
%s
}
`, stubInstallationID, stubAccountSlug, body)
}

// runStubApply applies body once and runs checks against the resulting state.
func runStubApply(t *testing.T, body string, checks ...resource.TestCheckFunc) *scmUnitStub {
	t.Helper()
	stub := newSCMUnitStub()
	stub.start(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: stubConfig(body),
			Check:  resource.ComposeAggregateTestCheckFunc(checks...),
		}},
	})
	return stub
}

// configuration_settings is Optional+Computed and so plans as unknown when omitted. A Go
// struct pointer cannot hold unknown, which used to fail the apply outright.
func TestScmUnitApply_MinimalConfig(t *testing.T) {
	runStubApply(t, ``,
		resource.TestCheckResourceAttr(stubResourceName, "account_id", stubAccountSlug),
		resource.TestCheckResourceAttr(stubResourceName, "installation_id", stubInstallationID),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.pr_summary_comment", "ALWAYS"),
	)
}

// adopt_existing is input-only, so it must land in state as null rather than unknown.
func TestScmUnitApply_AdoptExistingOmitted(t *testing.T) {
	runStubApply(t, `
  configuration_settings = {
    pr_summary_comment = "ALWAYS"
  }`,
		resource.TestCheckNoResourceAttr(stubResourceName, "adopt_existing"),
	)
}

func TestScmUnitApply_AdoptExistingSet(t *testing.T) {
	runStubApply(t, `
  adopt_existing = true`,
		resource.TestCheckResourceAttr(stubResourceName, "adopt_existing", "true"),
	)
}

// Documented clear-by-empty-list: state must keep [] rather than collapsing to null.
func TestScmUnitApply_EmptyArchiveConditionsStayEmpty(t *testing.T) {
	stub := runStubApply(t, `
  configuration_settings = {
    archive_conditions = []
  }`,
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.archive_conditions.#", "0"),
	)

	// The empty list must also reach the wire, or nothing is cleared server-side.
	reposConfig, ok := stub.lastPutConfigSettings(t)["installation_repositories_configuration"].(map[string]any)
	if !ok {
		t.Fatal("installation_repositories_configuration missing from the write")
	}
	archive, ok := reposConfig["archive_actions"].(map[string]any)
	if !ok {
		t.Fatalf("archive_actions omitted, so the clear would be ignored: %+v", reposConfig)
	}
	if conditions, ok := archive["conditions"].([]any); !ok || len(conditions) != 0 {
		t.Fatalf("expected archive conditions cleared to [], got: %+v", archive["conditions"])
	}
}

// An empty archive list next to a populated unavailable list must still clear the archive
// side; previously the empty side was dropped by omitempty and silently kept.
func TestScmUnitApply_AsymmetricConditionClear(t *testing.T) {
	stub := runStubApply(t, `
  configuration_settings = {
    archive_conditions     = []
    unavailable_conditions = ["DELETE_REPO"]
  }`,
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.archive_conditions.#", "0"),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.unavailable_conditions.#", "1"),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.unavailable_conditions.0", "DELETE_REPO"),
	)

	reposConfig := stub.lastPutConfigSettings(t)["installation_repositories_configuration"].(map[string]any)
	if _, ok := reposConfig["archive_actions"]; !ok {
		t.Fatalf("archive_actions must be sent even when empty: %+v", reposConfig)
	}
}

// Documented clear-by-empty-string for the project binding.
func TestScmUnitApply_EmptyProjectIDStaysEmpty(t *testing.T) {
	runStubApply(t, `
  project_id = ""`,
		resource.TestCheckResourceAttr(stubResourceName, "project_id", ""),
	)
}

// pr_summary_appendix = "" is the documented way to clear the appendix.
func TestScmUnitApply_EmptyPrSummaryAppendixStaysEmpty(t *testing.T) {
	runStubApply(t, `
  configuration_settings = {
    pr_summary_appendix = ""
  }`,
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.pr_summary_appendix", ""),
	)
}

// A populated config must round-trip unchanged.
func TestScmUnitApply_FullySpecifiedConfig(t *testing.T) {
	runStubApply(t, `
  adopt_existing    = true
  installation_mode = "SCAN_ALL_INCLUDE_FUTURE"
  default_policies  = true

  configuration_settings = {
    disable_scan_pull_requests = true
    comments_on_pull_requests  = "ONLY_ON_FAILED_ISSUES"
    pr_summary_comment         = "NEVER"
    skip_check_runs            = "ONLY_ON_INTERNAL_ISSUE"
    config_file_support        = "DISABLED"
    pr_summary_appendix        = "reviewed by terraform"
    archive_conditions         = ["AVOID_SCAN"]
    unavailable_conditions     = ["DELETE_REPO"]
  }`,
		resource.TestCheckResourceAttr(stubResourceName, "installation_mode", "SCAN_ALL_INCLUDE_FUTURE"),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.disable_scan_pull_requests", "true"),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.comments_on_pull_requests", "ONLY_ON_FAILED_ISSUES"),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.pr_summary_comment", "NEVER"),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.skip_check_runs", "ONLY_ON_INTERNAL_ISSUE"),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.config_file_support", "DISABLED"),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.pr_summary_appendix", "reviewed by terraform"),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.archive_conditions.0", "AVOID_SCAN"),
	)
}

// Updating one nested attribute must not disturb the others, and the second apply must
// still settle to an empty plan.
func TestScmUnitApply_UpdateSettlesWithoutDrift(t *testing.T) {
	stub := newSCMUnitStub()
	stub.start(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: stubConfig(`
  configuration_settings = {
    pr_summary_comment = "ALWAYS"
    archive_conditions = ["AVOID_SCAN"]
  }`),
				Check: resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.archive_conditions.0", "AVOID_SCAN"),
			},
			{
				Config: stubConfig(`
  configuration_settings = {
    pr_summary_comment = "NEVER"
    archive_conditions = []
  }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.pr_summary_comment", "NEVER"),
					resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.archive_conditions.#", "0"),
				),
			},
		},
	})
}
