package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	errFmtTestGotValue       = "got %+v"
	errFmtTestExpectedNilGot = "expected nil, " + errFmtTestGotValue
)

func TestListGroupAccessForGroup_FiltersByNestedGroupID(t *testing.T) {
	const targetGroupID = "g-target"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != apiRBACGroupAccessPath {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"total_items": 2,
			"data": []map[string]interface{}{
				{
					"id": "asg-other", "all_cloud_accounts": true,
					"group": map[string]string{"id": "g-other"},
					"role":  map[string]string{"id": "r1"},
				},
				{
					"id": "asg-want", "all_cloud_accounts": false,
					"group": map[string]string{"id": targetGroupID},
					"role":  map[string]string{"id": "r2"},
					"cloud_accounts": []map[string]string{
						{"id": "ca1"},
					},
					"user_filters":       []string{"f1"},
					"shiftleft_projects": []string{},
				},
			},
		})
	}))
	defer srv.Close()

	c := &APIClient{
		APIEndpoint: srv.URL,
		APIToken:    "tok",
		HTTPClient:  srv.Client(),
	}
	got, err := c.ListGroupAccessForGroup(targetGroupID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "asg-want" || got[0].GroupID != targetGroupID || got[0].RoleID != "r2" {
		t.Fatalf(errFmtTestGotValue, got)
	}
	if len(got[0].CloudAccounts) != 1 || got[0].CloudAccounts[0] != "ca1" {
		t.Fatalf("cloud accounts %+v", got[0].CloudAccounts)
	}
}

// Grant on page 2 — pagination must not stop at page 1.
func TestListGroupAccessForGroup_PagesUntilTotalItems(t *testing.T) {
	const targetGroupID = "g-target"

	var starts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start_at_index")
		starts = append(starts, start)
		if r.URL.Query().Get("limit") == "" {
			t.Fatalf("expected an explicit limit, got none")
		}
		var rows []map[string]interface{}
		if start == "0" || start == "" {
			rows = []map[string]interface{}{{
				"id": "asg-other", "group": map[string]string{"id": "g-other"},
				"role": map[string]string{"id": "r1"},
			}}
		} else {
			rows = []map[string]interface{}{{
				"id": "asg-want", "group": map[string]string{"id": targetGroupID},
				"role": map[string]string{"id": "r2"},
			}}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"total_items": 2,
			"data":        rows,
		})
	}))
	defer srv.Close()

	c := &APIClient{APIEndpoint: srv.URL, APIToken: "tok", HTTPClient: srv.Client()}
	got, err := c.ListGroupAccessForGroup(targetGroupID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "asg-want" {
		t.Fatalf("target row on a later page not returned: %+v", got)
	}
	if len(starts) < 2 || starts[1] != "1" {
		t.Fatalf("expected offset to advance by rows received, got starts=%v", starts)
	}
}

// No by-id URL; match by role+scope when id changed.
func TestFindGroupAccess_MatchesByScopeAcrossPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != apiRBACGroupAccessPath {
			t.Fatalf("unexpected path %s (by-id routes 404)", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"total_items": 1,
			"data": []map[string]interface{}{{
				"id": "asg-new", "all_cloud_accounts": false,
				"group":              map[string]string{"id": "g1"},
				"role":               map[string]string{"id": "r1"},
				"cloud_accounts":     []interface{}{},
				"user_filters":       []string{"bu1"},
				"shiftleft_projects": []string{},
			}},
		})
	}))
	defer srv.Close()

	c := &APIClient{APIEndpoint: srv.URL, APIToken: "tok", HTTPClient: srv.Client()}
	want := GroupAccess{
		GroupID:           "g1",
		RoleID:            "r1",
		AllCloudAccounts:  false,
		UserFilters:       []string{"bu1"},
		CloudAccounts:     []string{},
		ShiftleftProjects: []string{},
	}
	got, err := c.FindGroupAccess("stale-id", want)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != "asg-new" {
		t.Fatalf(errFmtTestGotValue, got)
	}
}

// Import: only assignment id known — scan whole collection.
func TestFindGroupAccess_ScansAllWhenGroupUnknown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"total_items": 2,
			"data": []map[string]interface{}{
				{"id": "asg-other", "group": map[string]string{"id": "g-other"}, "role": map[string]string{"id": "r9"}},
				{"id": "asg-import", "group": map[string]string{"id": "g1"}, "role": map[string]string{"id": "r1"}},
			},
		})
	}))
	defer srv.Close()

	c := &APIClient{APIEndpoint: srv.URL, APIToken: "tok", HTTPClient: srv.Client()}
	got, err := c.FindGroupAccess("asg-import", GroupAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != "asg-import" || got.GroupID != "g1" {
		t.Fatalf(errFmtTestGotValue, got)
	}
}

func TestFindGroupAccess_NotFoundReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"total_items": 0, "data": []interface{}{}})
	}))
	defer srv.Close()

	c := &APIClient{APIEndpoint: srv.URL, APIToken: "tok", HTTPClient: srv.Client()}
	got, err := c.FindGroupAccess("gone", GroupAccess{GroupID: "g1", RoleID: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf(errFmtTestExpectedNilGot, got)
	}
}

func TestCreateGroupAccess_ParsesWrappedDataID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != apiRBACGroupAccessPath {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"id":                 "asg-1",
				"group_id":           "g1",
				"role_id":            "r1",
				"all_cloud_accounts": false,
				"cloud_accounts":     []string{},
				"shiftleft_projects": []string{},
				"user_filters":       []string{"bu1"},
			},
		})
	}))
	defer srv.Close()

	c := &APIClient{
		APIEndpoint: srv.URL,
		APIToken:    "tok",
		HTTPClient:  srv.Client(),
	}
	got, err := c.CreateGroupAccess(GroupAccess{
		GroupID:           "g1",
		RoleID:            "r1",
		AllCloudAccounts:  false,
		UserFilters:       []string{"bu1"},
		CloudAccounts:     []string{},
		ShiftleftProjects: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "asg-1" || got.GroupID != "g1" || got.RoleID != "r1" {
		t.Fatalf(errFmtTestGotValue, got)
	}
}

func TestUpdateGroupAccess_UsesCollectionPathAndReReads(t *testing.T) {
	var putBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != apiRBACGroupAccessPath {
			t.Fatalf("unexpected path %s (by-id routes 404)", r.URL.Path)
		}
		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &putBody)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "data": map[string]interface{}{}})
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"total_items": 1,
				"data": []map[string]interface{}{{
					"id": "asg-1", "group": map[string]string{"id": "g1"},
					"role": map[string]string{"id": "r2"},
				}},
			})
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	c := &APIClient{APIEndpoint: srv.URL, APIToken: "tok", HTTPClient: srv.Client()}
	got, err := c.UpdateGroupAccess(GroupAccess{ID: "asg-1", GroupID: "g1", RoleID: "r2"})
	if err != nil {
		t.Fatal(err)
	}
	if putBody["id"] != "asg-1" {
		t.Fatalf("PUT body must carry id, got %+v", putBody)
	}
	if got == nil || got.ID != "asg-1" || got.RoleID != "r2" {
		t.Fatalf("re-read row wrong: %+v", got)
	}
}

func TestUpdateGroupAccess_RequiresID(t *testing.T) {
	c := &APIClient{APIEndpoint: "http://unused", APIToken: "t", HTTPClient: http.DefaultClient}
	_, err := c.UpdateGroupAccess(GroupAccess{})
	if err == nil || !strings.Contains(err.Error(), "id is required") {
		t.Fatalf("expected id required error, got %v", err)
	}
}

// Delete: id in body on collection path (by-id URL 404s).
func TestDeleteGroupAccess_UsesBodyNotByID(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "data": map[string]interface{}{}})
	}))
	defer srv.Close()

	c := &APIClient{APIEndpoint: srv.URL, APIToken: "tok", HTTPClient: srv.Client()}
	if err := c.DeleteGroupAccess("asg-1"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotPath != apiRBACGroupAccessPath {
		t.Fatalf("expected DELETE %s, got %s %s", apiRBACGroupAccessPath, gotMethod, gotPath)
	}
	if gotBody["id"] != "asg-1" {
		t.Fatalf("expected id in body, got %+v", gotBody)
	}
}

func TestDeleteGroupAccess_404Ignored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &APIClient{
		APIEndpoint: srv.URL,
		APIToken:    "tok",
		HTTPClient:  srv.Client(),
	}
	if err := c.DeleteGroupAccess("gone"); err != nil {
		t.Fatal(err)
	}
}
