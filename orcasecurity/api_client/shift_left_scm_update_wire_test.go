package api_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// capturedRequest records one HTTP call. updateScmUnit PUTs then re-lists, so
// captureServer (single slot) would lose the PUT body.
type capturedRequest struct {
	Method, Path string
	Body         map[string]any
}

func captureAllRequests(t *testing.T, responses map[string]string) (*APIClient, *[]capturedRequest) {
	t.Helper()
	reqs := &[]capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		*reqs = append(*reqs, capturedRequest{Method: r.Method, Path: r.URL.Path, Body: body})
		if resp, ok := responses[r.Method+" "+r.URL.Path]; ok {
			_, _ = w.Write([]byte(resp))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}, reqs
}

func findMethod(t *testing.T, reqs []capturedRequest, method string) capturedRequest {
	t.Helper()
	for _, r := range reqs {
		if r.Method == method {
			return r
		}
	}
	t.Fatalf("no %s request observed among %+v", method, reqs)
	return capturedRequest{}
}

// Assert Update* sends the expected JSON on the wire (not just struct fields).
func TestUpdateGithubInstallation_BodyShape(t *testing.T) {
	client, reqs := captureAllRequests(t, map[string]string{
		"GET " + githubInstallationsPath: `{"total_items":1,"data":[{"id":"inst-1","account_name":"acme"}]}`,
	})
	body := ScmInstallationUpdate{
		InstallationMode: "SELECTED_REPOSITORIES",
		DefaultPolicies:  false,
		Policies:         []string{"pol-1"},
		ConfigSettings: ShiftLeftConfigSettings{
			CommentsOnPullRequests: "ALWAYS",
			PrSummaryComment:       "ALWAYS",
			SkipCheckRuns:          "ALWAYS",
			ConfigFileSupport:      "ENABLED",
		},
	}
	if _, err := client.UpdateGithubInstallation("inst-1", body); err != nil {
		t.Fatal(err)
	}
	put := findMethod(t, *reqs, "PUT")
	if put.Path != "/api/shiftleft/github/installations/inst-1/" {
		t.Fatalf("wrong path: %s", put.Path)
	}
	if put.Body["installation_mode"] != "SELECTED_REPOSITORIES" {
		t.Errorf("installation_mode: %v", put.Body["installation_mode"])
	}
	policies, ok := put.Body["policies"].([]any)
	if !ok || len(policies) != 1 || policies[0] != "pol-1" {
		t.Errorf("policies: %v", put.Body["policies"])
	}
	if _, ok := put.Body["project_id"]; ok {
		t.Errorf("expected empty project_id omitted from the wire body, got %v", put.Body["project_id"])
	}
	cs, ok := put.Body["configuration_settings"].(map[string]any)
	if !ok {
		t.Fatalf("configuration_settings missing or wrong shape: %v", put.Body["configuration_settings"])
	}
	if cs["comments_on_pull_requests"] != "ALWAYS" {
		t.Errorf("configuration_settings.comments_on_pull_requests: %v", cs["comments_on_pull_requests"])
	}
}

func TestUpdateGithubInstallation_ProjectBoundOmitsPolicies(t *testing.T) {
	client, reqs := captureAllRequests(t, map[string]string{
		"GET " + githubInstallationsPath: `{"total_items":1,"data":[{"id":"inst-1","account_name":"acme"}]}`,
	})
	// Project-bound: policies must marshal as null (backend ignores while bound).
	body := ScmInstallationUpdate{
		InstallationMode: "SELECTED_REPOSITORIES",
		ProjectID:        "proj-1",
		Policies:         nil,
	}
	if _, err := client.UpdateGithubInstallation("inst-1", body); err != nil {
		t.Fatal(err)
	}
	put := findMethod(t, *reqs, "PUT")
	if put.Body["project_id"] != "proj-1" {
		t.Errorf("project_id: %v", put.Body["project_id"])
	}
	if v, ok := put.Body["policies"]; !ok || v != nil {
		t.Errorf("expected policies present and null when project-bound, got %v (present=%v)", v, ok)
	}
}

func TestUpdateGitlabGroup_BodyShape(t *testing.T) {
	client, reqs := captureAllRequests(t, map[string]string{
		"GET /api/shiftleft/gitlab/installations/inst-1/integrated_groups/": `{"total_items":1,"data":[{"id":"grp-1","account_name":"acme"}]}`,
	})
	body := ScmInstallationUpdate{
		InstallationMode: "SELECTED_REPOSITORIES",
		DefaultPolicies:  true,
		Policies:         []string{},
	}
	if _, err := client.UpdateGitlabGroup("inst-1", "grp-1", body); err != nil {
		t.Fatal(err)
	}
	put := findMethod(t, *reqs, "PUT")
	if put.Path != "/api/shiftleft/gitlab/installations/inst-1/integrated_group/grp-1/" {
		t.Fatalf("wrong path: %s", put.Path)
	}
	if put.Body["default_policies"] != true {
		t.Errorf("default_policies: %v", put.Body["default_policies"])
	}
	policies, ok := put.Body["policies"].([]any)
	if !ok || len(policies) != 0 {
		t.Errorf("expected an empty (not null) policies array when default_policies=true, got %v", put.Body["policies"])
	}
}

func TestUpdateBitbucketAccount_BodyShape(t *testing.T) {
	client, reqs := captureAllRequests(t, map[string]string{
		"GET /api/shiftleft/bitbucket/installations/inst-1/integrated_accounts/": `{"total_items":1,"data":[{"id":"acct-1","account_name":"acme"}]}`,
	})
	body := ScmInstallationUpdate{
		InstallationMode: "SELECTED_REPOSITORIES",
		ProjectID:        "",
		Policies:         []string{"pol-2"},
	}
	if _, err := client.UpdateBitbucketAccount("inst-1", "acct-1", body); err != nil {
		t.Fatal(err)
	}
	put := findMethod(t, *reqs, "PUT")
	if put.Path != "/api/shiftleft/bitbucket/installations/inst-1/integrated_accounts/acct-1/" {
		t.Fatalf("wrong path: %s", put.Path)
	}
	if _, ok := put.Body["project_id"]; ok {
		t.Errorf("expected empty project_id omitted from the wire body, got %v", put.Body["project_id"])
	}
}

func TestUpdateAzureDevopsAccount_BodyShape(t *testing.T) {
	client, reqs := captureAllRequests(t, map[string]string{
		"GET /api/shiftleft/azure_devops/installations/inst-1/integrated_accounts/": `{"total_items":1,"data":[{"id":"acct-1","account_name":"acme"}]}`,
	})
	body := ScmInstallationUpdate{
		InstallationMode: "SCAN_ALL_INCLUDE_FUTURE",
		ProjectID:        "proj-9",
	}
	if _, err := client.UpdateAzureDevopsAccount("inst-1", "acct-1", body); err != nil {
		t.Fatal(err)
	}
	put := findMethod(t, *reqs, "PUT")
	if put.Path != "/api/shiftleft/azure_devops/installations/inst-1/integrated_accounts/acct-1/" {
		t.Fatalf("wrong path: %s", put.Path)
	}
	if put.Body["project_id"] != "proj-9" {
		t.Errorf("project_id: %v", put.Body["project_id"])
	}
	if put.Body["installation_mode"] != "SCAN_ALL_INCLUDE_FUTURE" {
		t.Errorf("installation_mode: %v", put.Body["installation_mode"])
	}
}
