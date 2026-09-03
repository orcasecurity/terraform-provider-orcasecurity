package shift_left_policy_test

// attach_all_projects delegates the project set to the API instead of enumerating IDs. Driven
// against a stateful stub so the whole plan/apply loop is exercised: the request must carry
// attach_all_projects and no projects_ids, a settled set must plan clean, and a project appearing in the
// org afterwards must plan and apply a re-attach. Stub uses malicious_packages (no catalog).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

type attachAllStub struct {
	mu sync.Mutex
	// orgProjects is what GET /api/shiftleft/projects/ reports; attach_all_projects resolves against it.
	orgProjects []string
	attached    []string
	attachAlls  int
	// attachStatus is the status the projects PUT answers with.
	attachStatus int
	// lastBody is the decoded body of the most recent projects PUT.
	lastBody map[string]any
}

func (s *attachAllStub) addOrgProject(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orgProjects = append(s.orgProjects, id)
}

func (s *attachAllStub) snapshot() (orgProjects, attached []string, attachAlls int, lastBody map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.orgProjects...), append([]string(nil), s.attached...), s.attachAlls, s.lastBody
}

func (s *attachAllStub) policyBody() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	projects := make([]map[string]any, 0, len(s.attached))
	for _, id := range s.attached {
		projects = append(projects, map[string]any{"id": id})
	}
	return map[string]any{
		"id": stubPolicyID, "name": "tf-attach-all", "type": "malicious_packages",
		"builtin": false, "disabled": false, "warn_mode": false,
		"priority_failure_threshold": "HIGH", "projects": projects,
	}
}

func (s *attachAllStub) handleProjectsPut(w http.ResponseWriter, r *http.Request) {
	if s.attachStatus != http.StatusOK {
		http.Error(w, `{"detail":"project attach rejected"}`, s.attachStatus)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Mirror PolicyProjectsSerializer.validate: both keys together is a 400, and so is neither,
	// because projects_ids is only optional while attach_all_projects is true. A regression to the
	// pre-rename attach_all key lands in the second branch — an unknown key is dropped, leaving a
	// body the endpoint reads as empty — so a stale key fails loudly instead of detaching all.
	_, hasIDs := body["projects_ids"]
	attachAll := body["attach_all_projects"] == true
	if attachAll && hasIDs {
		http.Error(w, `{"errors":{"projects_ids":["projects_ids must not be provided when attach_all_projects is True."]}}`, http.StatusBadRequest)
		return
	}
	if !attachAll && !hasIDs {
		http.Error(w, `{"errors":{"projects_ids":["This field is required."]}}`, http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.lastBody = body
	switch {
	case attachAll:
		s.attachAlls++
		s.attached = append([]string(nil), s.orgProjects...)
	case hasIDs:
		ids, _ := body["projects_ids"].([]any)
		s.attached = nil
		for _, id := range ids {
			s.attached = append(s.attached, id.(string))
		}
	}
	s.mu.Unlock()

	_ = json.NewEncoder(w).Encode(s.policyBody())
}

// projectsList answers the paged org project list ModifyPlan reads to detect a stale set.
func (s *attachAllStub) projectsList(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	all := append([]string(nil), s.orgProjects...)
	s.mu.Unlock()

	start, _ := strconv.Atoi(r.URL.Query().Get("start_at_index"))
	data := make([]map[string]any, 0, len(all))
	if start < len(all) {
		for _, id := range all[start:] {
			data = append(data, map[string]any{"id": id, "name": id, "key": id})
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"total_items": len(all), "data": data})
}

func (s *attachAllStub) start(t *testing.T) {
	t.Helper()
	const base = "/api/shiftleft/malicious_packages/policies/"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/shiftleft/projects/":
			s.projectsList(w, r)
		case r.Method == http.MethodPost && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(s.policyBody())
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/projects/"):
			s.handleProjectsPut(w, r)
		case r.Method == http.MethodPut && r.URL.Path == base+stubPolicyID+"/":
			_ = json.NewEncoder(w).Encode(s.policyBody())
		case r.Method == http.MethodGet && r.URL.Path == base+stubPolicyID+"/":
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

const attachAllConfig = orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_policy" "all" {
  type                       = "malicious_packages"
  name                       = "tf-attach-all"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"
  attach_all_projects        = true
}
`

func TestShiftLeftPolicyAttachAll_CreateSendsAttachAllAndRecordsEveryProject(t *testing.T) {
	stub := &attachAllStub{attachStatus: http.StatusOK, orgProjects: []string{"proj-1", "proj-2", "proj-3"}}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: attachAllConfig,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.all", "attach_all_projects", "true"),
				resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.all", "projects_ids.#", "3"),
				resource.TestCheckTypeSetElemAttr("orcasecurity_shift_left_policy.all", "projects_ids.*", "proj-2"),
			),
		}},
	})

	_, attached, attachAlls, lastBody := stub.snapshot()
	if attachAlls != 1 {
		t.Errorf("expected exactly one attach_all call on create, got %d", attachAlls)
	}
	if len(attached) != 3 {
		t.Errorf("expected all 3 org projects attached, got %v", attached)
	}
	if _, present := lastBody["projects_ids"]; present {
		t.Errorf("attach_all request must not enumerate projects_ids, got %#v", lastBody)
	}
}

// A settled attach_all must not churn: no new project means no planned change and no extra PUT.
func TestShiftLeftPolicyAttachAll_SettledSetPlansClean(t *testing.T) {
	stub := &attachAllStub{attachStatus: http.StatusOK, orgProjects: []string{"proj-1", "proj-2"}}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: attachAllConfig},
			{
				Config:   attachAllConfig,
				PlanOnly: true,
			},
		},
	})

	if _, _, attachAlls, _ := stub.snapshot(); attachAlls != 1 {
		t.Errorf("a settled attach_all must not re-attach on refresh, got %d attach_all calls", attachAlls)
	}
}

// The whole point of attach_all on every apply: a project added in Orca afterwards gets picked up.
func TestShiftLeftPolicyAttachAll_NewOrgProjectPlansAndAppliesReattach(t *testing.T) {
	stub := &attachAllStub{attachStatus: http.StatusOK, orgProjects: []string{"proj-1"}}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: attachAllConfig,
				Check:  resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.all", "projects_ids.#", "1"),
			},
			{
				// Someone creates a project in Orca between applies.
				PreConfig: func() { stub.addOrgProject("proj-2") },
				Config:    attachAllConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("orcasecurity_shift_left_policy.all", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_shift_left_policy.all", "projects_ids.#", "2"),
					resource.TestCheckTypeSetElemAttr("orcasecurity_shift_left_policy.all", "projects_ids.*", "proj-2"),
				),
			},
		},
	})

	_, attached, attachAlls, _ := stub.snapshot()
	if attachAlls != 2 {
		t.Errorf("expected a second attach_all once a new project appeared, got %d", attachAlls)
	}
	if len(attached) != 2 {
		t.Errorf("expected both projects attached after the second apply, got %v", attached)
	}
}

// Rollback still applies on the attach_all path — a failed attach must not leave an untracked policy.
func TestShiftLeftPolicyAttachAll_FailedAttachDeletesPolicy(t *testing.T) {
	stub := &attachAllStub{attachStatus: http.StatusInternalServerError, orgProjects: []string{"proj-1"}}
	stub.start(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      attachAllConfig,
			ExpectError: regexp.MustCompile(`Error setting AppSec policy projects`),
		}},
	})
}
