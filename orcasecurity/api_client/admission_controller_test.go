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

func TestGetAdmissionControllerControl_MultipleMatches(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":[
				{"id":"ctrl-1","name":"first","template_id":"tpl-1",
				 "cluster_scope":{"kinds":[{"apiGroups":[""],"kinds":["Pod"],"versions":[""]}]}},
				{"id":"ctrl-2","name":"second","template_id":"tpl-1",
				 "cluster_scope":{"kinds":[{"apiGroups":[""],"kinds":["Pod"],"versions":[""]}]}}]}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	control, err := client.GetAdmissionControllerControl("ctrl-1")
	if err == nil {
		t.Fatalf("expected error for multiple matches, got control %+v", control)
	}
	if control != nil {
		t.Errorf("expected nil control on multiple matches, got %+v", control)
	}
	if !strings.Contains(err.Error(), "got 2") {
		t.Errorf("expected error to report match count, got: %v", err)
	}
}

// assertControlEmptyOptionalFieldsPayload verifies the serialization of unset
// optional control fields: description present as explicit null, nil
// input_parameters sent as explicit {}, empty apiGroups/versions inside
// cluster_scope kinds omitted.
func assertControlEmptyOptionalFieldsPayload(t *testing.T, body []byte) {
	t.Helper()
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("payload not JSON: %v", err)
	}
	if desc, has := payload["description"]; !has || desc != nil {
		t.Errorf("nil description must be sent as explicit null: %s", body)
	}
	if params, has := payload["input_parameters"]; !has {
		t.Errorf("nil input_parameters must be sent as {}, not omitted (PUT omit retains the remote value): %s", body)
	} else if obj, ok := params.(map[string]interface{}); !ok || len(obj) != 0 {
		t.Errorf("nil input_parameters must be sent as {}: %s", body)
	}
	scope := payload["cluster_scope"].(map[string]interface{})
	kind := scope["kinds"].([]interface{})[0].(map[string]interface{})
	if _, has := kind["apiGroups"]; has {
		t.Errorf("empty apiGroups must be omitted from payload: %s", body)
	}
	if _, has := kind["versions"]; has {
		t.Errorf("empty versions must be omitted from payload: %s", body)
	}
	if kinds := kind["kinds"].([]interface{}); len(kinds) != 1 || kinds[0] != "Pod" {
		t.Errorf("unexpected kinds: %s", body)
	}
}

// Pins the serialization contract for optional fields. The API's PUT routes
// are full-replace, but an omitted key *retains* the remote value while an
// explicit null/{}/[] clears it — so description (null when unset) and
// input_parameters ({} when unset) must always be present. Empty
// apiGroups/versions inside cluster_scope kinds are still omitted: the
// backend replaces cluster_scope wholesale, so nested keys don't retain.
func TestCreateAdmissionControllerControl_EmptyOptionalFields(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		body, _ := io.ReadAll(req.Body)
		assertControlEmptyOptionalFieldsPayload(t, body)
		return &http.Response{
			StatusCode: 201,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":{
				"id":"ctrl-new","name":"minimal control","template_id":"tpl-1","template_name":"k8sallowedrepos",
				"cluster_scope":{"kinds":[{"kinds":["Pod"]}]}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	control, err := client.CreateAdmissionControllerControl(AdmissionControllerControl{
		Name:       "minimal control",
		TemplateID: "tpl-1",
		ClusterScope: AdmissionControllerClusterScope{Kinds: []AdmissionControllerClusterScopeKind{
			{APIGroups: []string{}, Kinds: []string{"Pod"}, Versions: []string{}},
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if control.ID != "ctrl-new" {
		t.Errorf("expected ctrl-new, got %s", control.ID)
	}
}

// Same contract on the update path: clearing input_parameters relies on the
// client sending {} — an omitted key would silently retain the remote value.
func TestUpdateAdmissionControllerControl_EmptyOptionalFields(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "PUT" || req.URL.Path != "/api/admission_controller/controls/ctrl-1" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		body, _ := io.ReadAll(req.Body)
		assertControlEmptyOptionalFieldsPayload(t, body)
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":{
				"id":"ctrl-1","name":"minimal control","template_id":"tpl-1","template_name":"k8sallowedrepos",
				"cluster_scope":{"kinds":[{"kinds":["Pod"]}]}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	_, err := client.UpdateAdmissionControllerControl(AdmissionControllerControl{
		ID:         "ctrl-1",
		Name:       "minimal control",
		TemplateID: "tpl-1",
		ClusterScope: AdmissionControllerClusterScope{Kinds: []AdmissionControllerClusterScopeKind{
			{APIGroups: []string{}, Kinds: []string{"Pod"}, Versions: []string{}},
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAdmissionControllerPolicy(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.URL.Path != "/api/admission_controller/policies/pol-1" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":{
				"id":"pol-1","name":"my policy","description":"","is_active":true,
				"enforcement_action":"monitor","controls":["ctrl-1"],"scopes":[],
				"stale_relations":{"controls":{"deleted_count":0}},"user":"someone"}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.GetAdmissionControllerPolicy("pol-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy == nil || policy.ID != "pol-1" || !policy.IsActive || policy.EnforcementAction != "monitor" {
		t.Fatalf("unexpected policy: %+v", policy)
	}
	if len(policy.Controls) != 1 || policy.Controls[0] != "ctrl-1" {
		t.Errorf("unexpected controls: %+v", policy.Controls)
	}
}

func TestGetAdmissionControllerPolicy_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"status":"failure","error_code":"not_found","message":"No Policy matches the given query."}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.GetAdmissionControllerPolicy("missing")
	if err != nil {
		t.Fatalf("expected nil error for 404, got: %v", err)
	}
	if policy != nil {
		t.Errorf("expected nil policy, got %+v", policy)
	}
}

func TestCreateAdmissionControllerPolicy_OmitsScopesKey(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		body, _ := io.ReadAll(req.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("payload not JSON: %v", err)
		}
		if _, hasScopes := payload["scopes"]; hasScopes {
			t.Errorf("policy payload must not include scopes key: %s", body)
		}
		if payload["enforcement_action"] != "block" {
			t.Errorf("unexpected enforcement_action: %v", payload["enforcement_action"])
		}
		return &http.Response{
			StatusCode: 201,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":{
				"id":"pol-new","name":"my policy","is_active":true,
				"enforcement_action":"block","controls":["ctrl-1"]}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.CreateAdmissionControllerPolicy(AdmissionControllerPolicy{
		Name: "my policy", IsActive: true, EnforcementAction: "block", Controls: []string{"ctrl-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.ID != "pol-new" {
		t.Errorf("expected pol-new, got %s", policy.ID)
	}
}

func TestUpdateAdmissionControllerPolicy(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		// PATCH, not PUT: the PUT route rejects a policy's own unchanged name.
		if req.Method != "PATCH" || req.URL.Path != "/api/admission_controller/policies/pol-1" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":{
				"id":"pol-1","name":"renamed","is_active":false,
				"enforcement_action":"monitor","controls":[]}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	policy, err := client.UpdateAdmissionControllerPolicy(AdmissionControllerPolicy{
		ID: "pol-1", Name: "renamed", IsActive: false, EnforcementAction: "monitor", Controls: []string{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Name != "renamed" || policy.IsActive {
		t.Errorf("unexpected policy: %+v", policy)
	}
}

func TestDeleteAdmissionControllerPolicy(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "DELETE" || req.URL.Path != "/api/admission_controller/policies/pol-1" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(``))}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	if err := client.DeleteAdmissionControllerPolicy("pol-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAdmissionControllerScope(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.URL.Path != "/api/admission_controller/scopes/scope-1" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":{
				"id":"scope-1","name":"my assignment","description":"",
				"cloud_accounts":[],"clusters":["cluster-1"],"full_organization":false,
				"policies":[{"id":"pol-1","name":"my policy"}],
				"stale_relations":{"cloud_accounts":{"deleted_count":0}}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	scope, err := client.GetAdmissionControllerScope("scope-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope == nil || scope.ID != "scope-1" || scope.FullOrganization {
		t.Fatalf("unexpected scope: %+v", scope)
	}
	if len(scope.Policies) != 1 || scope.Policies[0].ID != "pol-1" {
		t.Errorf("unexpected embedded policies: %+v", scope.Policies)
	}
	if len(scope.Clusters) != 1 || scope.Clusters[0] != "cluster-1" {
		t.Errorf("unexpected clusters: %+v", scope.Clusters)
	}
}

func TestCreateAdmissionControllerScope_SendsPolicyIDs(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		body, _ := io.ReadAll(req.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("payload not JSON: %v", err)
		}
		if _, hasPolicies := payload["policies"]; hasPolicies {
			t.Errorf("scope payload must not include read-only policies key: %s", body)
		}
		ids := payload["policy_ids"].([]interface{})
		if len(ids) != 1 || ids[0] != "pol-1" {
			t.Errorf("unexpected policy_ids: %s", body)
		}
		if payload["full_organization"] != true {
			t.Errorf("expected full_organization true: %s", body)
		}
		return &http.Response{
			StatusCode: 201,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":{
				"id":"scope-new","name":"my assignment","cloud_accounts":[],"clusters":[],
				"full_organization":true,"policies":[{"id":"pol-1","name":"my policy"}]}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	scope, err := client.CreateAdmissionControllerScope(AdmissionControllerScope{
		Name: "my assignment", FullOrganization: true,
		CloudAccounts: []string{}, Clusters: []string{}, PolicyIDs: []string{"pol-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope.ID != "scope-new" {
		t.Errorf("expected scope-new, got %s", scope.ID)
	}
}

func TestUpdateAdmissionControllerScope(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "PUT" || req.URL.Path != "/api/admission_controller/scopes/scope-1" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		body, _ := io.ReadAll(req.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("payload not JSON: %v", err)
		}
		// PUT is full-replace with omitted-key-retains semantics: an empty
		// policy_ids must be sent as [] (detach all), never dropped, and a nil
		// description as explicit null (clear).
		if ids, has := payload["policy_ids"]; !has {
			t.Errorf("empty policy_ids must be sent as [], not omitted: %s", body)
		} else if list, ok := ids.([]interface{}); !ok || len(list) != 0 {
			t.Errorf("unexpected policy_ids: %s", body)
		}
		if desc, has := payload["description"]; !has || desc != nil {
			t.Errorf("nil description must be sent as explicit null: %s", body)
		}
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":{
				"id":"scope-1","name":"renamed","cloud_accounts":[],"clusters":[],
				"full_organization":true,"policies":[]}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	scope, err := client.UpdateAdmissionControllerScope(AdmissionControllerScope{
		ID: "scope-1", Name: "renamed", FullOrganization: true,
		CloudAccounts: []string{}, Clusters: []string{}, PolicyIDs: []string{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope.Name != "renamed" {
		t.Errorf("expected renamed, got %s", scope.Name)
	}
}

func TestDeleteAdmissionControllerScope(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "DELETE" || req.URL.Path != "/api/admission_controller/scopes/scope-1" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(``))}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	if err := client.DeleteAdmissionControllerScope("scope-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAdmissionControllerTemplates(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.URL.Path != "/api/admission_controller/templates" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{"status":"success","data":[
				{"id":"tpl-1","name":"k8sallowedrepos","source":"internal","controller_type":"gatekeeper",
				 "display_name":"Allowed Container Registries","version":"1.1.1",
				 "description":"desc","supported_kinds":["Pod"]}]}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	templates, err := client.GetAdmissionControllerTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 1 || templates[0].Name != "k8sallowedrepos" || templates[0].DisplayName != "Allowed Container Registries" {
		t.Errorf("unexpected templates: %+v", templates)
	}
}
