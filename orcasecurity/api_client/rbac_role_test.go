package api_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRbacRoleListAPIResponseUnmarshal(t *testing.T) {
	raw := `{"status":"success","data":[{"id":"a","name":"Viewer"},{"id":"b","name":"Editor"}]}`
	var parsed rbacRoleListAPIResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Status != "success" || len(parsed.Data) != 2 {
		t.Fatalf("got %+v", parsed)
	}
	if parsed.Data[0].ID != "a" || parsed.Data[0].Name != "Viewer" {
		t.Fatalf("first role %+v", parsed.Data[0])
	}
}

func TestListRBACRoles_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/rbac/role" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"success","data":[{"id":"r1","name":"Viewer"},{"id":"r2","name":"Editor"}]}`))
	}))
	defer srv.Close()

	c := &APIClient{
		APIEndpoint: srv.URL,
		APIToken:    "tok",
		HTTPClient:  srv.Client(),
	}
	roles, err := c.ListRBACRoles()
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 2 || roles[0].Name != "Viewer" || roles[1].ID != "r2" {
		t.Fatalf("got %+v", roles)
	}
}

func TestListRBACRoles_NonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"failed","data":[]}`))
	}))
	defer srv.Close()

	c := &APIClient{
		APIEndpoint: srv.URL,
		APIToken:    "tok",
		HTTPClient:  srv.Client(),
	}
	_, err := c.ListRBACRoles()
	if err == nil {
		t.Fatal("expected error")
	}
}
