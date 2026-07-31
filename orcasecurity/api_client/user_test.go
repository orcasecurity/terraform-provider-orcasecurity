package api_client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListUsers_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("start_at_index"); got != "0" {
			t.Fatalf("start_at_index %s", got)
		}
		_, _ = w.Write([]byte(`{"status":"success","total_items":2,"data":[
			{"user_id":"u1","email":"a@example.com","first":"A","last":"One","status":"active","mfa_required":true,"mfa_enabled":false},
			{"user_id":"u2","email":"b@example.com","first":"B","last":"Two","status":"active","mfa_required":false,"mfa_enabled":true}
		]}`))
	}))
	defer srv.Close()

	c := &APIClient{APIEndpoint: srv.URL, APIToken: "tok", HTTPClient: srv.Client()}
	users, err := c.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].Email != "a@example.com" || users[1].ID != "u2" {
		t.Fatalf("got %+v", users)
	}
}

func TestListUsers_Paginates(t *testing.T) {
	pages := [][]string{
		{`{"user_id":"u1","email":"a@example.com"}`},
		{`{"user_id":"u2","email":"b@example.com"}`},
		{},
	}
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rows := pages[call]
		call++
		body := fmt.Sprintf(`{"status":"success","total_items":2,"data":[%s]}`, joinCommaUser(rows))
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := &APIClient{APIEndpoint: srv.URL, APIToken: "tok", HTTPClient: srv.Client()}
	users, err := c.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].ID != "u1" || users[1].ID != "u2" {
		t.Fatalf("got %+v", users)
	}
}

func joinCommaUser(rows []string) string {
	out := ""
	for i, r := range rows {
		if i > 0 {
			out += ","
		}
		out += r
	}
	return out
}

func TestListUsers_NonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"failed","data":[]}`))
	}))
	defer srv.Close()

	c := &APIClient{APIEndpoint: srv.URL, APIToken: "tok", HTTPClient: srv.Client()}
	if _, err := c.ListUsers(); err == nil {
		t.Fatal("expected error")
	}
}
