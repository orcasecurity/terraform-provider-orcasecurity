package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
		t.Fatalf("got %+v", got)
	}
}

func TestGetGroupAccess_404ReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"not found"}`)
	}))
	defer srv.Close()

	c := &APIClient{
		APIEndpoint: srv.URL,
		APIToken:    "tok",
		HTTPClient:  srv.Client(),
	}
	got, err := c.GetGroupAccess("missing-id")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestGetGroupAccess_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/rbac/access/group/asg-1" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(groupAccessAPIResponseType{
			Data: GroupAccess{
				ID:                "asg-1",
				GroupID:           "g1",
				RoleID:            "r1",
				AllCloudAccounts:  false,
				UserFilters:       []string{"f1"},
				CloudAccounts:     []string{},
				ShiftleftProjects: []string{},
			},
		})
	}))
	defer srv.Close()

	c := &APIClient{
		APIEndpoint: srv.URL,
		APIToken:    "tok",
		HTTPClient:  srv.Client(),
	}
	got, err := c.GetGroupAccess("asg-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != "asg-1" {
		t.Fatalf("got %+v", got)
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

func TestUpdateGroupAccess_RequiresID(t *testing.T) {
	c := &APIClient{APIEndpoint: "http://unused", APIToken: "t", HTTPClient: http.DefaultClient}
	_, err := c.UpdateGroupAccess(GroupAccess{})
	if err == nil || !strings.Contains(err.Error(), "id is required") {
		t.Fatalf("expected id required error, got %v", err)
	}
}
