package api_client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHasBuiltinPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policies []ShiftLeftProjectPolicy
		want     bool
	}{
		{"no policies", nil, false},
		{"only custom policies", []ShiftLeftProjectPolicy{{ID: "pol-1", Builtin: false}}, false},
		{"builtin present", []ShiftLeftProjectPolicy{{ID: "pol-1", Builtin: false}, {ID: "pol-2", Builtin: true}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasBuiltinPolicy(tt.policies); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// The API never echoes the write-only default_policies flag; GetShiftLeftProject
// must reconstruct it from the attached policies so Read does not drift to false.
func TestGetShiftLeftProject_ReconstructsDefaultPolicies(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "GET", "/api/shiftleft/projects/proj-1/")
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"proj-1","name":"Project 1","key":"project-1","policies":[{"id":"pol-1","builtin":true},{"id":"pol-2","builtin":false}]}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	project, err := client.GetShiftLeftProject("proj-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project == nil || project.ID != "proj-1" {
		t.Fatalf("expected proj-1, got %+v", project)
	}
	if !project.DefaultPolicies {
		t.Error("default_policies must be reconstructed as true when a builtin policy is attached")
	}
	if len(project.Policies) != 2 || project.Policies[0].ID != "pol-1" {
		t.Errorf("unexpected policies: %+v", project.Policies)
	}
}

func TestGetShiftLeftProject_NoBuiltinPolicies(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"proj-1","name":"Project 1","key":"project-1","policies":[{"id":"pol-2","builtin":false}]}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	project, err := client.GetShiftLeftProject("proj-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.DefaultPolicies {
		t.Error("default_policies must be false when no builtin policy is attached")
	}
}

// Pins the error-or-value contract: GetShiftLeftProject never returns
// (nil, nil) — non-OK responses (including 404) surface as errors.
func TestGetShiftLeftProject_NotFoundIsError(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"detail":"not found"}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	project, err := client.GetShiftLeftProject("missing")
	if err == nil {
		t.Fatal("expected an error on 404")
	}
	if project != nil {
		t.Errorf("expected nil project on 404, got %+v", project)
	}
}
