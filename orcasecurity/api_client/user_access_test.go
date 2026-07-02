package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const testUserAccessID = "b8f4cbab-3a06-4bb6-aac9-84ae98a64725"
const testUserAccessUserID = "9d710fef-da7e-4072-944d-e359f45f201d"
const testUserAccessRoleID = "9234cabf-9d29-4972-b445-35917e728bea"

const testUserAccessListResponse = `{
	"status": "success",
	"data": [
		{
			"id": "b8f4cbab-3a06-4bb6-aac9-84ae98a64725",
			"all_cloud_accounts": true,
			"cloud_accounts": [],
			"role": {"id": "9234cabf-9d29-4972-b445-35917e728bea", "name": "Administrator"},
			"user": {"id": "9d710fef-da7e-4072-944d-e359f45f201d", "email": "a@orca.security"},
			"user_filters": [],
			"shiftleft_projects": []
		},
		{
			"id": "other-assignment",
			"all_cloud_accounts": false,
			"cloud_accounts": [{"id": "ca-1", "name": "acc"}],
			"role": {"id": "role-x", "name": "Viewer"},
			"user": {"id": "another-user", "email": "b@orca.security"},
			"user_filters": [],
			"shiftleft_projects": []
		}
	]
}`

func TestCreateUserAccess(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method == "POST" {
			if !strings.HasSuffix(req.URL.Path, "/api/rbac/access/user") {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			body, _ := io.ReadAll(req.Body)
			payload := map[string]interface{}{}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("invalid JSON body: %v", err)
			}
			if payload["user_id"] != testUserAccessUserID {
				t.Errorf("expected user_id %s, got %v", testUserAccessUserID, payload["user_id"])
			}
			if payload["role_id"] != testUserAccessRoleID {
				t.Errorf("expected role_id %s, got %v", testUserAccessRoleID, payload["role_id"])
			}
			return &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"id":"` + testUserAccessID + `"}}`)),
				Request:    req,
			}
		}
		// GET (FindUserAccess refresh)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(testUserAccessListResponse)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	ua, err := apiClient.CreateUserAccess(UserAccess{
		UserID:           testUserAccessUserID,
		RoleID:           testUserAccessRoleID,
		AllCloudAccounts: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ua.ID != testUserAccessID {
		t.Errorf("unexpected id: %s", ua.ID)
	}
}

func TestFindUserAccess(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "GET" {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if req.URL.Query().Get("user_id") != testUserAccessUserID {
			t.Errorf("expected user_id query, got %s", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(testUserAccessListResponse)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	ua, err := apiClient.FindUserAccess(testUserAccessID, UserAccess{UserID: testUserAccessUserID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ua == nil {
		t.Fatal("expected an assignment, got nil")
	}
	if ua.ID != testUserAccessID || ua.RoleID != testUserAccessRoleID || !ua.AllCloudAccounts {
		t.Errorf("unexpected assignment: %+v", ua)
	}
}

func TestFindUserAccess_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":[]}`)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	ua, err := apiClient.FindUserAccess(testUserAccessID, UserAccess{UserID: testUserAccessUserID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ua != nil {
		t.Errorf("expected nil, got %+v", ua)
	}
}

func TestDeleteUserAccess(t *testing.T) {
	called := false
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		called = true
		if req.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		body, _ := io.ReadAll(req.Body)
		payload := map[string]string{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("delete body is not JSON: %v", err)
		}
		if payload["id"] != testUserAccessID {
			t.Errorf("expected id %s in body, got %v", testUserAccessID, payload)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"status":"success"}`)), Request: req}
	})}

	apiClient := newTestAPIClient(httpClient)
	if err := apiClient.DeleteUserAccess(testUserAccessID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("delete request was not made")
	}
}
