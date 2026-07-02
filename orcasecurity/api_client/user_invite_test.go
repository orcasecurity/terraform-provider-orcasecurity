package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const testUserInviteID = "2636ad80-fdeb-4698-9231-0eabb11b48a6"

func TestCreateUserInvite(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "POST" {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/api/user_invites/bulk_create/") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		body, _ := io.ReadAll(req.Body)
		payload := map[string]interface{}{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid JSON body: %v", err)
		}
		emails, ok := payload["invite_user_emails"].([]interface{})
		if !ok || len(emails) != 1 || emails[0] != "tf@example.com" {
			t.Errorf("expected single invite email, got %v", payload["invite_user_emails"])
		}
		return &http.Response{
			StatusCode: 201,
			Body: io.NopCloser(strings.NewReader(
				`[{"id":"` + testUserInviteID + `","email":"tf@example.com","invite_link":"https://app.orca/invite/x","expired":false}]`)),
			Request: req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	invite, err := apiClient.CreateUserInvite(UserInviteRequest{
		InviteUserEmails: []string{"tf@example.com"},
		RoleID:           testUserAccessRoleID,
		ShouldSendEmail:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invite.ID != testUserInviteID {
		t.Errorf("unexpected id: %s", invite.ID)
	}
	if invite.InviteLink == "" {
		t.Error("expected invite_link to be populated")
	}
}

func TestGetUserInvite(t *testing.T) {
	body := `{"results":[{"id":"` + testUserInviteID + `","email":"tf@example.com","expired":false}]}`
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "GET" {
			t.Errorf("expected GET, got %s", req.Method)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Request: req}
	})}

	apiClient := newTestAPIClient(httpClient)
	invite, err := apiClient.GetUserInvite(testUserInviteID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invite == nil || invite.Email != "tf@example.com" {
		t.Errorf("unexpected invite: %+v", invite)
	}
}

func TestGetUserInvite_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results":[]}`)), Request: req}
	})}

	apiClient := newTestAPIClient(httpClient)
	invite, err := apiClient.GetUserInvite(testUserInviteID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invite != nil {
		t.Errorf("expected nil, got %+v", invite)
	}
}

func TestDeleteUserInvite(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.Path, "/api/user_invites/"+testUserInviteID) {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader("")), Request: req}
	})}

	apiClient := newTestAPIClient(httpClient)
	if err := apiClient.DeleteUserInvite(testUserInviteID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
