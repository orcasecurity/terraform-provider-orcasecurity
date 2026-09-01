package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const testCrownJewelGroupID = "tf-wasp-1553-probe-do-not-keep"

func TestGetCrownJewel(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "GET", "/api/attack_paths/crown_jewels")
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`[
				{"group_unique_id":"vm_other","severity":4,"description":"keep me"},
				{"group_unique_id":"` + testCrownJewelGroupID + `","severity":4,"description":"tf-probe","create_time":"2026-09-01T14:00:57+03:00","update_time":"2026-09-01T14:00:57+03:00","user_email":"lab@example.com","last_user_action":"set_as_crown_jewel"}
			]`)),
			Request: req,
		}
	})}

	client := newTestAPIClient(httpClient)
	jewel, err := client.GetCrownJewel(testCrownJewelGroupID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jewel == nil {
		t.Fatal("expected crown jewel, got nil")
	}
	if jewel.GroupUniqueID != testCrownJewelGroupID {
		t.Errorf("unexpected group_unique_id: %s", jewel.GroupUniqueID)
	}
	if jewel.Description != "tf-probe" {
		t.Errorf("unexpected description: %s", jewel.Description)
	}
}

func TestGetCrownJewel_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`[{"group_unique_id":"vm_other","description":"keep me"}]`)),
			Request:    req,
		}
	})}

	client := newTestAPIClient(httpClient)
	jewel, err := client.GetCrownJewel("missing")
	if err != nil {
		t.Fatalf("expected nil error when missing so the resource can RemoveResource, got: %v", err)
	}
	if jewel != nil {
		t.Errorf("expected nil crown jewel when missing, got %+v", jewel)
	}
}

func TestCreateCrownJewel(t *testing.T) {
	var postedBody []byte
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		switch req.Method {
		case "POST":
			assertMethodPath(t, req, "POST", "/api/attack_paths/crown_jewels")
			postedBody, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"status":"success"}`)),
				Request:    req,
			}
		case "GET":
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(strings.NewReader(`[
					{"group_unique_id":"` + testCrownJewelGroupID + `","severity":4,"description":"tf-probe","last_user_action":"set_as_crown_jewel"}
				]`)),
				Request: req,
			}
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil
		}
	})}

	client := newTestAPIClient(httpClient)
	jewel, err := client.CreateCrownJewel(CrownJewel{
		GroupUniqueID: testCrownJewelGroupID,
		Description:   "tf-probe",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jewel == nil || jewel.GroupUniqueID != testCrownJewelGroupID || jewel.Description != "tf-probe" {
		t.Errorf("unexpected crown jewel: %+v", jewel)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(postedBody, &payload); err != nil {
		t.Fatalf("invalid POST body: %v", err)
	}
	ids, _ := payload["group_unique_ids"].([]interface{})
	if len(ids) != 1 || ids[0] != testCrownJewelGroupID {
		t.Errorf("expected group_unique_ids [%s], got %v", testCrownJewelGroupID, payload["group_unique_ids"])
	}
	if payload["description"] != "tf-probe" {
		t.Errorf("unexpected description in payload: %v", payload["description"])
	}
}

func TestUpdateCrownJewel(t *testing.T) {
	var postedBody []byte
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		switch req.Method {
		case "POST":
			postedBody, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"status":"success"}`)),
				Request:    req,
			}
		case "GET":
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(strings.NewReader(`[
					{"group_unique_id":"` + testCrownJewelGroupID + `","description":"tf-probe-updated"}
				]`)),
				Request: req,
			}
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil
		}
	})}

	client := newTestAPIClient(httpClient)
	jewel, err := client.UpdateCrownJewel(CrownJewel{
		GroupUniqueID: testCrownJewelGroupID,
		Description:   "tf-probe-updated",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jewel == nil || jewel.Description != "tf-probe-updated" {
		t.Errorf("unexpected crown jewel: %+v", jewel)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(postedBody, &payload); err != nil {
		t.Fatalf("invalid POST body: %v", err)
	}
	if payload["description"] != "tf-probe-updated" {
		t.Errorf("unexpected description in payload: %v", payload["description"])
	}
}

func TestDeleteCrownJewel(t *testing.T) {
	var deletedBody []byte
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "DELETE", "/api/attack_paths/crown_jewels")
		deletedBody, _ = io.ReadAll(req.Body)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success"}`)),
			Request:    req,
		}
	})}

	client := newTestAPIClient(httpClient)
	if err := client.DeleteCrownJewel(testCrownJewelGroupID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(deletedBody, &payload); err != nil {
		t.Fatalf("invalid DELETE body: %v", err)
	}
	ids, _ := payload["group_unique_ids"].([]interface{})
	if len(ids) != 1 || ids[0] != testCrownJewelGroupID {
		t.Errorf("expected group_unique_ids [%s], got %v", testCrownJewelGroupID, payload["group_unique_ids"])
	}
}
