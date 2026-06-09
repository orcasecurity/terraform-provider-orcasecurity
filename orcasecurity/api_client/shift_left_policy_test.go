package api_client

import (
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

func TestDoesShiftLeftPolicyExist(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "HEAD" {
			t.Errorf("expected HEAD, got %s", req.Method)
		}
		if req.URL.Path != "/api/shiftleft/iac/policies/policy-123/" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(``)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := client.DoesShiftLeftPolicyExist("iac", "policy-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected policy to exist for 200 response")
	}
}

func TestDoesShiftLeftPolicyExist_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(``)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := client.DoesShiftLeftPolicyExist("iac", "missing")
	if err != nil {
		t.Fatalf("expected no error on 404 so the plan recreates the resource, got: %v", err)
	}
	if exists {
		t.Error("expected policy not to exist for 404 response")
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
