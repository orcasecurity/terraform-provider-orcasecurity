package shift_left_scm_posture_default_policy_test

// Stub-driven coverage of the adopt/write path. The resource PUTs the org-wide
// built-in singleton, so the two behaviors that matter most are (1) an apply
// that leaves controls unset must hydrate the PUT with the live overrides
// rather than wiping them, and (2) a live policy_data the provider cannot
// decode must abort before the PUT ever happens. The live acceptance test
// covers the same flows but is double-gated (TF_ACC + explicit opt-in), so
// this stub is what runs in ordinary CI.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type postureStub struct {
	mu         sync.Mutex
	disabled   bool
	policyData json.RawMessage
	putBodies  []map[string]any
}

func (s *postureStub) puts() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]map[string]any(nil), s.putBodies...)
}

func (s *postureStub) policyJSON() map[string]any {
	var data any
	if len(s.policyData) > 0 {
		// Malformed fixtures cannot round-trip through json.Marshal, so pass
		// them along as-is via RawMessage.
		data = s.policyData
	}
	return map[string]any{
		"id":          "posture-pol-1",
		"name":        "Default SCM Posture Policy",
		"description": "built-in",
		"disabled":    s.disabled,
		"policy_data": data,
	}
}

func (s *postureStub) handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/shiftleft/scm_posture/policy/" {
		http.Error(w, "unexpected path "+r.URL.Path, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	s.mu.Lock()
	defer s.mu.Unlock()
	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(s.policyJSON())
	case http.MethodPut:
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.putBodies = append(s.putBodies, body)
		if disabled, ok := body["disabled"].(bool); ok {
			s.disabled = disabled
		}
		if data, err := json.Marshal(body["policy_data"]); err == nil {
			s.policyData = data
		}
		_ = json.NewEncoder(w).Encode(s.policyJSON())
	default:
		http.Error(w, "unexpected method "+r.Method, http.StatusInternalServerError)
	}
}

func (s *postureStub) start(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(srv.Close)
	t.Setenv("TF_ACC", "1")
	t.Setenv("ORCASECURITY_API_ENDPOINT", srv.URL)
	t.Setenv("ORCASECURITY_API_TOKEN", "stub-token")
	// The gate exists to protect the real org-wide singleton; the stub owns
	// nothing shared, so make sure an unset gate cannot skip these tests and
	// a set one cannot leak lab credentials into them.
	t.Setenv("ORCA_TEST_SCM_POSTURE_DEFAULT_ALLOW", "")
}

func putControls(t *testing.T, body map[string]any) []any {
	t.Helper()
	data, ok := body["policy_data"].(map[string]any)
	if !ok {
		t.Fatalf("PUT body missing policy_data: %+v", body)
	}
	controls, ok := data["controls"].([]any)
	if !ok {
		t.Fatalf("policy_data.controls must be an explicit array, got: %+v", data["controls"])
	}
	return controls
}

// An apply that only touches `disabled` must carry the live control overrides
// through the PUT unchanged; the endpoint replaces policy_data wholesale, so
// sending [] here would wipe every override in the org.
func TestScmPostureDefaultPolicyApply_UnsetControlsPreserveLiveOverrides(t *testing.T) {
	stub := &postureStub{
		disabled:   false,
		policyData: json.RawMessage(`{"controls":[{"id":"c1","priority":"HIGH"},{"id":"c2","disabled":true}]}`),
	}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_scm_posture_default_policy" "t" {
  disabled = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_scm_posture_default_policy.t", "id", "posture-pol-1"),
					resource.TestCheckResourceAttr("orcasecurity_shift_left_scm_posture_default_policy.t", "disabled", "true"),
					resource.TestCheckNoResourceAttr("orcasecurity_shift_left_scm_posture_default_policy.t", "controls.#"),
				),
			},
		},
	})

	puts := stub.puts()
	if len(puts) != 1 {
		t.Fatalf("expected exactly one PUT, got %d: %+v", len(puts), puts)
	}
	controls := putControls(t, puts[0])
	if len(controls) != 2 {
		t.Fatalf("live overrides must be hydrated into the PUT, got: %+v", controls)
	}
	first, _ := controls[0].(map[string]any)
	if first["id"] != "c1" || first["priority"] != "HIGH" {
		t.Fatalf("hydrated override mismapped: %+v", first)
	}
}

// controls = [] is the documented way to clear all overrides: it must reach the
// wire as an explicit empty array, not be dropped or sent as null.
func TestScmPostureDefaultPolicyApply_EmptyControlsClearOverrides(t *testing.T) {
	stub := &postureStub{
		policyData: json.RawMessage(`{"controls":[{"id":"c1","priority":"HIGH"}]}`),
	}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_scm_posture_default_policy" "t" {
  controls = []
}`,
				Check: resource.TestCheckResourceAttr("orcasecurity_shift_left_scm_posture_default_policy.t", "controls.#", "0"),
			},
		},
	})

	puts := stub.puts()
	if len(puts) == 0 {
		t.Fatal("expected a PUT clearing the overrides")
	}
	if controls := putControls(t, puts[len(puts)-1]); len(controls) != 0 {
		t.Fatalf("expected controls cleared to [], got: %+v", controls)
	}
}

// When the live policy_data cannot be decoded, the write must abort before the
// PUT: proceeding would send empty controls and silently wipe every live
// override org-wide (the failure mode the restore path in the acceptance test
// guards against). The fixture is valid JSON of the wrong shape — truly
// malformed JSON never gets this far because the transport decode rejects it.
func TestScmPostureDefaultPolicyApply_MalformedLiveDataAbortsBeforePut(t *testing.T) {
	stub := &postureStub{
		policyData: json.RawMessage(`{"controls": {"not": "an array"}}`),
	}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_scm_posture_default_policy" "t" {
  disabled = true
}`,
				ExpectError: regexp.MustCompile(`could not decode live policy_data`),
			},
		},
	})

	if puts := stub.puts(); len(puts) != 0 {
		t.Fatalf("decode failure must abort before the PUT, but got %d PUT(s): %+v", len(puts), puts)
	}
}
