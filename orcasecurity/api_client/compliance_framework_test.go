package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func testAPIClient(fn RoundTripFunc) APIClient {
	return APIClient{
		APIEndpoint: "http://localhost",
		APIToken:    "secret",
		HTTPClient:  &http.Client{Transport: fn},
	}
}

// jsonStatus is the api_client-test copy of testutils.JSONResponse.
// Resource packages use testutils; this package cannot import it without a cycle.
func jsonStatus(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestGetComplianceFrameworkSelections_OptionalKeys(t *testing.T) {
	client := testAPIClient(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "GET", "/api/compliance/frameworks/select")
		return jsonStatus(200, `{
			"minimal": {
				"id": "minimal",
				"active": false,
				"selection_scopes": [],
				"display_name": "Minimal",
				"custom": true
			},
			"full": {
				"id": "full",
				"active": true,
				"selection_scopes": ["user", "organization"],
				"display_name": "Full",
				"custom": false,
				"description": "desc",
				"type": "Orca Frameworks",
				"version": "1.0.0",
				"version_agnostic_display_name": "Full",
				"is_ready": true,
				"framework_cloud_vendors": ["aws"],
				"icon_family": "orca",
				"orca_end_of_support_date": null,
				"visibility": "Organizational",
				"is_forced_cloud_vendors": false
			}
		}`)
	})

	got, err := client.GetComplianceFrameworkSelections()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	min := got["minimal"]
	if min.ID != "minimal" || min.Active || min.Custom != true || min.DisplayName != "Minimal" {
		t.Errorf("minimal entry: %+v", min)
	}
	if min.Type != nil || min.Version != nil || min.IsReady != nil || min.Visibility != nil {
		t.Errorf("minimal optional keys must be nil, got %+v", min)
	}
	if min.SelectionScopes == nil || len(min.SelectionScopes) != 0 {
		t.Errorf("minimal selection_scopes must be empty slice, got %#v", min.SelectionScopes)
	}
	full := got["full"]
	if full.Type == nil || *full.Type != "Orca Frameworks" {
		t.Errorf("full type: %+v", full.Type)
	}
	if full.IsReady == nil || !*full.IsReady {
		t.Errorf("full is_ready: %+v", full.IsReady)
	}
	if len(full.SelectionScopes) != 2 {
		t.Errorf("full scopes: %#v", full.SelectionScopes)
	}
}

func TestSelectAndDeselectComplianceFrameworks(t *testing.T) {
	var posts, deletes int
	var lastBody map[string]interface{}
	client := testAPIClient(func(req *http.Request) *http.Response {
		body, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(body, &lastBody)
		switch req.Method {
		case "POST":
			posts++
			assertMethodPath(t, req, "POST", "/api/compliance/frameworks/select")
		case "DELETE":
			deletes++
			assertMethodPath(t, req, "DELETE", "/api/compliance/frameworks/select")
		default:
			t.Errorf("unexpected method %s", req.Method)
		}
		return jsonStatus(200, `{}`)
	})

	if err := client.SelectComplianceFrameworks([]string{"cis_aws"}, "user"); err != nil {
		t.Fatal(err)
	}
	if lastBody["scope"] != "user" {
		t.Errorf("select scope: %v", lastBody["scope"])
	}
	ids, _ := lastBody["framework_ids"].([]interface{})
	if len(ids) != 1 || ids[0] != "cis_aws" {
		t.Errorf("select ids: %v", lastBody["framework_ids"])
	}

	if err := client.DeselectComplianceFrameworks([]string{"cis_aws"}, "organization"); err != nil {
		t.Fatal(err)
	}
	if lastBody["scope"] != "organization" {
		t.Errorf("deselect scope: %v", lastBody["scope"])
	}
	if posts != 1 || deletes != 1 {
		t.Errorf("posts=%d deletes=%d", posts, deletes)
	}
}

func TestGetComplianceFramework_NotFoundVsServerError(t *testing.T) {
	t.Run("404 is gone", func(t *testing.T) {
		client := testAPIClient(func(req *http.Request) *http.Response {
			assertMethodPath(t, req, "GET", "/api/compliance/frameworks/missing")
			return jsonStatus(404, `{"error":"Framework missing not found."}`)
		})
		fw, err := client.GetCustomComplianceFramework("missing")
		if err != nil {
			t.Fatalf("404 must be (nil, nil), got err %v", err)
		}
		if fw != nil {
			t.Errorf("404 must be (nil, nil), got %+v", fw)
		}
	})
	t.Run("500 is an error", func(t *testing.T) {
		client := testAPIClient(func(req *http.Request) *http.Response {
			return jsonStatus(500, `{"error":"boom"}`)
		})
		fw, err := client.GetCustomComplianceFramework("x")
		if err == nil {
			t.Fatal("500 must be an error, not gone")
		}
		if fw != nil {
			t.Errorf("500 must not return a framework, got %+v", fw)
		}
	})
}

func TestGetComplianceCatalogFramework(t *testing.T) {
	client := testAPIClient(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "GET", "/api/compliance/catalog/3887")
		return jsonStatus(200, `{
			"data": {
				"total_frameworks": 1,
				"frameworks": [{
					"name": "Lab",
					"display_name": "Lab",
					"framework_id": "3887",
					"custom": true,
					"sections": [
						{"id": "1", "name": "Parent", "total_tests": 0, "tests": [],
						 "sections": [
							{"id": "1.1", "name": "Child One", "total_tests": 1,
							 "tests": [{"rule_id": "r1", "reference_id": "1.1.1", "priority": "Medium", "cis_level": ["Level 1"]}]}
						 ]}
					]
				}]
			}
		}`)
	})
	fw, err := client.GetComplianceCatalogFramework("3887")
	if err != nil || fw == nil {
		t.Fatalf("got %+v, %v", fw, err)
	}
	if fw.FrameworkID != "3887" || len(fw.Sections) != 1 {
		t.Fatalf("unexpected framework: %+v", fw)
	}
	if fw.Sections[0].Sections == nil || fw.Sections[0].Sections[0].Tests[0].ReferenceID != "1.1.1" {
		t.Errorf("nested tree: %+v", fw.Sections)
	}
	got := fw.Sections[0].Sections[0].Tests[0].CISLevel
	if len(got) != 1 || got[0] != "Level 1" {
		t.Errorf("cis_level must decode as a string list, got %#v", got)
	}
}

func TestDeleteCustomComplianceFramework_404Ignored(t *testing.T) {
	client := testAPIClient(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "DELETE", "/api/compliance/frameworks/3887")
		return jsonStatus(404, `{"error":"Framework 3887 not found."}`)
	})
	if err := client.DeleteCustomComplianceFramework("3887"); err != nil {
		t.Fatalf("404 on delete must be ignored, got %v", err)
	}
}

func TestDeleteCustomComplianceFramework_OtherErrorSurfaced(t *testing.T) {
	client := testAPIClient(func(req *http.Request) *http.Response {
		return jsonStatus(500, `{"error":"boom"}`)
	})
	if err := client.DeleteCustomComplianceFramework("3887"); err == nil {
		t.Fatal("non-404 delete error must surface")
	}
}

func TestCreateCustomComplianceFramework_OmitsCheckedKeys(t *testing.T) {
	client := testAPIClient(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "POST", "/api/compliance/frameworks")
		body, _ := io.ReadAll(req.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("body: %v", err)
		}
		if _, ok := payload["checkedKeys"]; ok {
			t.Errorf("checkedKeys must not be sent, got %s", body)
		}
		if payload["name"] != "Lab" {
			t.Errorf("name: %v", payload["name"])
		}
		return jsonStatus(200, `{"data":{"id": 3887, "name":"Lab","description":""}}`)
	})
	got, err := client.CreateCustomComplianceFramework(CustomComplianceFrameworkRequest{
		Name: "Lab",
		Sections: []CustomComplianceFrameworkSection{{
			Name:     "Flat",
			Tests:    []CustomComplianceFrameworkTest{{RuleID: "r1", RuleIDInFramework: "1.1"}},
			Sections: []CustomComplianceFrameworkSection{},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID.String() != "3887" {
		t.Errorf("id: %s", got.ID)
	}
}
