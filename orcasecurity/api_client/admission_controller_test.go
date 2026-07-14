package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetAdmissionControllerControl(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "GET" {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/api/admission_controller/controls" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		if req.URL.Query().Get("ids") != "ctrl-1" {
			t.Errorf("unexpected ids param: %s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":[{
				"id":"ctrl-1","name":"my control","description":"desc",
				"template_id":"tpl-1","template_name":"k8sallowedrepos",
				"cluster_scope":{"kinds":[{"apiGroups":[""],"kinds":["Pod"],"versions":[""]}]},
				"input_parameters":{"repos":["docker.io/library"]}}]}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	control, err := client.GetAdmissionControllerControl("ctrl-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if control == nil || control.ID != "ctrl-1" {
		t.Fatalf("expected control ctrl-1, got %+v", control)
	}
	if control.TemplateName != "k8sallowedrepos" {
		t.Errorf("unexpected template_name: %s", control.TemplateName)
	}
	if len(control.ClusterScope.Kinds) != 1 || control.ClusterScope.Kinds[0].Kinds[0] != "Pod" {
		t.Errorf("unexpected cluster_scope: %+v", control.ClusterScope)
	}
	if !json.Valid(control.InputParameters) {
		t.Errorf("input_parameters not valid JSON: %s", control.InputParameters)
	}
}

func TestGetAdmissionControllerControl_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":[]}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	control, err := client.GetAdmissionControllerControl("missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if control != nil {
		t.Errorf("expected nil for missing control, got %+v", control)
	}
}

func TestCreateAdmissionControllerControl(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "POST" {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/api/admission_controller/controls" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		body, _ := io.ReadAll(req.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("payload not JSON: %v", err)
		}
		if _, hasID := payload["id"]; hasID {
			t.Errorf("create payload must not carry id: %s", body)
		}
		scope := payload["cluster_scope"].(map[string]interface{})
		kind := scope["kinds"].([]interface{})[0].(map[string]interface{})
		if _, ok := kind["apiGroups"]; !ok {
			t.Errorf("expected camelCase apiGroups key, got: %s", body)
		}
		return &http.Response{
			StatusCode: 201,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":{
				"id":"ctrl-new","name":"my control","template_id":"tpl-1","template_name":"k8sallowedrepos",
				"cluster_scope":{"kinds":[{"apiGroups":[""],"kinds":["Pod"],"versions":[""]}]},
				"input_parameters":{"repos":["docker.io/library"]}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	control, err := client.CreateAdmissionControllerControl(AdmissionControllerControl{
		Name:       "my control",
		TemplateID: "tpl-1",
		ClusterScope: AdmissionControllerClusterScope{Kinds: []AdmissionControllerClusterScopeKind{
			{APIGroups: []string{""}, Kinds: []string{"Pod"}, Versions: []string{""}},
		}},
		InputParameters: json.RawMessage(`{"repos":["docker.io/library"]}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if control.ID != "ctrl-new" {
		t.Errorf("expected ctrl-new, got %s", control.ID)
	}
}

func TestUpdateAdmissionControllerControl(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "PUT" {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		if req.URL.Path != "/api/admission_controller/controls/ctrl-1" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":{
				"id":"ctrl-1","name":"renamed","template_id":"tpl-2","template_name":"k8sdisallowinteractivetty",
				"cluster_scope":{"kinds":[{"apiGroups":[""],"kinds":["Pod"],"versions":[""]}]},
				"input_parameters":{}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	control, err := client.UpdateAdmissionControllerControl(AdmissionControllerControl{
		ID:         "ctrl-1",
		Name:       "renamed",
		TemplateID: "tpl-2",
		ClusterScope: AdmissionControllerClusterScope{Kinds: []AdmissionControllerClusterScopeKind{
			{APIGroups: []string{""}, Kinds: []string{"Pod"}, Versions: []string{""}},
		}},
		InputParameters: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if control.Name != "renamed" {
		t.Errorf("expected renamed, got %s", control.Name)
	}
}

func TestDeleteAdmissionControllerControl(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if req.URL.Path != "/api/admission_controller/controls/ctrl-1" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(``))}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	if err := client.DeleteAdmissionControllerControl("ctrl-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
