package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestShiftLeftPolicyTypePath(t *testing.T) {
	if ShiftLeftPolicyTypePath("scm_posture") != "scm_posture" {
		t.Errorf("expected scm_posture path segment")
	}
	if ShiftLeftPolicyTypePath("iac") != "iac" {
		t.Errorf("expected iac path segment")
	}
}

func TestGetShiftLeftPolicy(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "GET" {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/api/shiftleft/iac/policies/policy-123/" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"policy-123","name":"test","type":"iac","disabled":false,"warn_mode":false,"priority_failure_threshold":"HIGH"}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.GetShiftLeftPolicy("iac", "policy-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy == nil || policy.ID != "policy-123" {
		t.Errorf("expected policy id policy-123, got %+v", policy)
	}
}

func TestCreateShiftLeftPolicy_ScmPosture(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.URL.Path != "/api/shiftleft/scm_posture/policies/" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(`{"id":"scm-1","name":"scm policy","type":"scm_posture"}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.CreateShiftLeftPolicy("scm_posture", ShiftLeftPolicy{Name: "scm policy"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.ID != "scm-1" {
		t.Errorf("expected scm-1, got %s", policy.ID)
	}
}

func TestDeleteShiftLeftPolicy(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		return &http.Response{
			StatusCode: 204,
			Body:       io.NopCloser(strings.NewReader(``)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	err := client.DeleteShiftLeftPolicy("container_image", "ci-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGetShiftLeftPolicy_NotFoundReturnsNil pins the drift contract: a 404
// yields (nil, nil) so the resource Read removes the policy from state and the
// plan recreates it, rather than surfacing an error.
func TestGetShiftLeftPolicy_NotFoundReturnsNil(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "GET" {
			t.Errorf("reads must use GET (HEAD 5xxes on some policy types), got %s", req.Method)
		}
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(``)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.GetShiftLeftPolicy("iac", "missing")
	if err != nil {
		t.Fatalf("expected no error on 404 so the plan recreates the resource, got: %v", err)
	}
	if policy != nil {
		t.Errorf("expected nil policy for 404 response, got %+v", policy)
	}
}

// TestGetShiftLeftPolicy_ServerErrorIsError pins that a transient 5xx is an
// error (surfaced as a diagnostic), never mistaken for "policy deleted".
func TestGetShiftLeftPolicy_ServerErrorIsError(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader(`{"error":"boom"}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.GetShiftLeftPolicy("iac", "policy-123")
	if err == nil {
		t.Fatal("expected an error for a 500 response")
	}
	if policy != nil {
		t.Errorf("expected nil policy alongside the error, got %+v", policy)
	}
}

func TestUpdateShiftLeftPolicy(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "PUT" {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if req.URL.Path != "/api/shiftleft/iac/policies/policy-123/" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"policy-123","name":"updated","type":"iac"}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.UpdateShiftLeftPolicy("iac", "policy-123", ShiftLeftPolicy{Name: "updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Name != "updated" {
		t.Errorf("expected name updated, got %s", policy.Name)
	}
}

// TestShiftLeftPolicy_ProjectsIdsFromProjects is the RED/GREEN case for the
// populateProjectsIds helper: the live GET response returns attached projects
// as `"projects":[{"id":...}]`, never `"projects_ids"`, so ProjectsIds must be
// derived from Projects after unmarshal.
func TestShiftLeftPolicy_ProjectsIdsFromProjects(t *testing.T) {
	body := []byte(`{"id":"p1","name":"OSS Licenses Policy","builtin":true,
		"projects":[{"id":"proj-a"},{"id":"proj-b"}]}`)
	var p ShiftLeftPolicy
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatal(err)
	}
	p.populateProjectsIds()
	if len(p.ProjectsIds) != 2 || p.ProjectsIds[0] != "proj-a" || p.ProjectsIds[1] != "proj-b" {
		t.Fatalf("expected [proj-a proj-b], got %v", p.ProjectsIds)
	}
}

func TestShiftLeftPolicy_ProjectsIdsPrefersExplicit(t *testing.T) {
	// If the API ever returns projects_ids directly, don't clobber it.
	p := ShiftLeftPolicy{ProjectsIds: []string{"x"}}
	p.populateProjectsIds()
	if len(p.ProjectsIds) != 1 || p.ProjectsIds[0] != "x" {
		t.Fatalf("explicit projects_ids overwritten: %v", p.ProjectsIds)
	}
}

// TestGetShiftLeftPolicy_PopulatesProjectsIdsFromProjects exercises the full
// client wiring: GetShiftLeftPolicy must populate ProjectsIds from the
// `projects` array of a realistic GET response, not just the bare helper.
func TestGetShiftLeftPolicy_PopulatesProjectsIdsFromProjects(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{"id":"policy-123","name":"OSS Licenses Policy","type":"licenses",
				"builtin":true,"disabled":false,"warn_mode":false,"priority_failure_threshold":"HIGH",
				"projects":[{"id":"proj-a","name":"Project A"},{"id":"proj-b","name":"Project B"}]}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.GetShiftLeftPolicy("licenses", "policy-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
	if len(policy.ProjectsIds) != 2 || policy.ProjectsIds[0] != "proj-a" || policy.ProjectsIds[1] != "proj-b" {
		t.Fatalf("expected ProjectsIds [proj-a proj-b], got %v", policy.ProjectsIds)
	}
}

func TestGetShiftLeftPolicyCatalogControls(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.URL.Path != "/api/shiftleft/iac/catalog/controls" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"controls":[{"id":"ctrl-1","title":"Control 1"}]}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	catalog, err := client.GetShiftLeftPolicyCatalogControls("iac")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if catalog.Body == nil {
		t.Error("expected catalog body in response")
	}
}
