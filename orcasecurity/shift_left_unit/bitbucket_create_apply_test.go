package shift_left_unit_test

// In-process SCM-unit stub applies (shared AdoptedUnitOps path); Bitbucket as stand-in.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	stubInstallationID = "11111111-1111-1111-1111-111111111111"
	stubOrcaAccountID  = "22222222-2222-2222-2222-222222222222"
	stubAccountSlug    = "acme-workspace"
	stubResourceName   = "orcasecurity_shift_left_bitbucket_account.test"
)

// Stub: PUT replaces configuration_settings wholesale; later reads return stored values.
type scmUnitStub struct {
	mu   sync.Mutex
	unit map[string]any
	puts []map[string]any

	// integrated: false simulates a unit that doesn't exist yet — GET misses until a POST
	// integrate call (adopt_existing must be irrelevant on that path, since Create takes the
	// fresh-integrate branch, not adopt). Defaults to true: the account already exists in Orca,
	// so Create must adopt it (adopt_existing = true required).
	integrated bool
	integrates []map[string]any

	// volatileSideEffects: opt-in server-side moves of Computed fields (scan-all, project rebind).
	volatileSideEffects bool
}

func newSCMUnitStub() *scmUnitStub {
	return &scmUnitStub{
		integrated: true,
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

// newNotYetIntegratedSCMUnitStub starts with the account undiscovered: GET misses until the
// resource POSTs a fresh integrate, which is the only Create path that can leave adopt_existing
// unset (adopting an existing account always requires it).
func newNotYetIntegratedSCMUnitStub() *scmUnitStub {
	stub := newSCMUnitStub()
	stub.integrated = false
	return stub
}

func (s *scmUnitStub) isIntegrated() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.integrated
}

func (s *scmUnitStub) applyIntegrate(body map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.integrates = append(s.integrates, body)
	s.integrated = true

	if mode, ok := body["installation_mode"].(string); ok && mode != "" {
		s.unit["installation_mode"] = mode
	}
	if defaultPolicies, ok := body["default_policies"].(bool); ok {
		s.unit["default_policies"] = defaultPolicies
	}
	if cfg, ok := body["configuration_settings"].(map[string]any); ok {
		s.unit["configuration_settings"] = cfg
	}
	if projectID, ok := body["project_id"].(string); ok && projectID != "" {
		s.unit["project"] = map[string]any{"id": projectID}
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
		priorMode, _ := s.unit["installation_mode"].(string)
		s.unit["installation_mode"] = mode
		if s.volatileSideEffects && mode == "SCAN_ALL_INCLUDE_FUTURE" {
			s.unit["integrated_repositories_count"] = 37
			s.unit["scan_all_state"] = "RUNNING"
		}
		// Leaving legacy scan-all halts the enrollment flow server-side.
		if s.volatileSideEffects && priorMode == "SCAN_ALL" && mode != "SCAN_ALL_INCLUDE_FUTURE" {
			s.unit["scan_all_state"] = "STOPPED"
			s.unit["integrated_repositories_count"] = 5
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

func (s *scmUnitStub) start(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		isUnitPath := strings.Contains(r.URL.Path, "/integrated_accounts/")
		isIntegratePath := strings.Contains(r.URL.Path, "/integrated_repositories/")
		switch {
		case r.Method == http.MethodPut && isUnitPath:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.applyPut(body)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		case r.Method == http.MethodPost && isIntegratePath:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.applyIntegrate(body)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		case r.Method == http.MethodGet && isUnitPath:
			if !s.isIntegrated() {
				body, _ := json.Marshal(map[string]any{"total_items": 0, "data": []map[string]any{}})
				_, _ = w.Write([]byte(testutils.FirstPageOnly(r, string(body))))
				return
			}
			body, _ := json.Marshal(map[string]any{"total_items": 1, "data": []map[string]any{s.snapshot()}})
			_, _ = w.Write([]byte(testutils.FirstPageOnly(r, string(body))))
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

func runStubApply(t *testing.T, body string, checks ...resource.TestCheckFunc) *scmUnitStub {
	t.Helper()
	return runStubApplyOn(t, newSCMUnitStub(), body, checks...)
}

func runStubApplyOn(t *testing.T, stub *scmUnitStub, body string, checks ...resource.TestCheckFunc) *scmUnitStub {
	t.Helper()
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

// configuration_settings must travel as types.Object (can hold unknown). The account already
// exists on the stub, so adopting it requires adopt_existing = true.
func TestScmUnitApply_MinimalConfig(t *testing.T) {
	runStubApply(t, `
  adopt_existing = true`,
		resource.TestCheckResourceAttr(stubResourceName, "account_id", stubAccountSlug),
		resource.TestCheckResourceAttr(stubResourceName, "installation_id", stubInstallationID),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.pr_summary_comment", "ALWAYS"),
	)
}

// adopt_existing is input-only, so it must land in state as null rather than unknown. Only a
// genuine fresh integrate can omit it and still succeed (adopting an existing account always
// requires it), so this runs against a not-yet-integrated stub — installation_mode must be
// SCAN_ALL_INCLUDE_FUTURE, since the integrate guard rejects SELECTED_REPOSITORIES on that path.
func TestScmUnitApply_AdoptExistingOmitted(t *testing.T) {
	runStubApplyOn(t, newNotYetIntegratedSCMUnitStub(), `
  installation_mode = "SCAN_ALL_INCLUDE_FUTURE"
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

func TestScmUnitApply_EmptyArchiveConditionsStayEmpty(t *testing.T) {
	stub := runStubApply(t, `
  adopt_existing = true
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

// Asymmetric clear must still send the empty archive side.
func TestScmUnitApply_AsymmetricConditionClear(t *testing.T) {
	stub := runStubApply(t, `
  adopt_existing = true
  configuration_settings = {
    archive_conditions     = []
    unavailable_conditions = ["DELETE_REPO"]
  }`,
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.archive_conditions.#", "0"),
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.unavailable_conditions.#", "1"),
		resource.TestCheckTypeSetElemAttr(stubResourceName, "configuration_settings.unavailable_conditions.*", "DELETE_REPO"),
	)

	reposConfig := stub.lastPutConfigSettings(t)["installation_repositories_configuration"].(map[string]any)
	if _, ok := reposConfig["archive_actions"]; !ok {
		t.Fatalf("archive_actions must be sent even when empty: %+v", reposConfig)
	}
}

func TestScmUnitApply_EmptyProjectIDStaysEmpty(t *testing.T) {
	runStubApply(t, `
  adopt_existing = true
  project_id     = ""`,
		resource.TestCheckResourceAttr(stubResourceName, "project_id", ""),
	)
}

func TestScmUnitApply_EmptyPrSummaryAppendixStaysEmpty(t *testing.T) {
	runStubApply(t, `
  adopt_existing = true
  configuration_settings = {
    pr_summary_appendix = ""
  }`,
		resource.TestCheckResourceAttr(stubResourceName, "configuration_settings.pr_summary_appendix", ""),
	)
}

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
		resource.TestCheckTypeSetElemAttr(stubResourceName, "configuration_settings.archive_conditions.*", "AVOID_SCAN"),
	)
}

func TestScmUnitApply_UpdateSettlesWithoutDrift(t *testing.T) {
	stub := newSCMUnitStub()
	stub.start(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: stubConfig(`
  adopt_existing = true
  configuration_settings = {
    pr_summary_comment = "ALWAYS"
    archive_conditions = ["AVOID_SCAN"]
  }`),
				Check: resource.TestCheckTypeSetElemAttr(stubResourceName, "configuration_settings.archive_conditions.*", "AVOID_SCAN"),
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
