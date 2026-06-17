package api_client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

const testDiscoveryViewID = "e19babce-cb90-428c-a108-af63914b03e7"

const testDiscoveryViewResponse = `{
	"data": {
		"preference_id": "e19babce-cb90-428c-a108-af63914b03e7",
		"name": "orca-disco-view-inventory-by-account",
		"view_type": "discovery",
		"organization_level": true,
		"filter_data": {
			"query2": {
				"models": ["Inventory"],
				"type": "object_set"
			}
		},
		"extra_params": {
			"sort2": "-OrcaScore",
			"groupBy2": [],
			"columns2": {
				"keys": ["CloudAccount", "OrcaScore"]
			}
		}
	}
}`

func TestDoesDiscoveryViewExist_Found(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != http.MethodHead {
			t.Errorf("expected HEAD, got %s", req.Method)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	exists, err := apiClient.DoesDiscoveryViewExist(testDiscoveryViewID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected discovery view to exist")
	}
}

func TestDoesDiscoveryViewExist_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	exists, err := apiClient.DoesDiscoveryViewExist("invalid-id")
	if err != nil {
		t.Fatalf("expected no error on 404 so the resource can be removed from state, got: %v", err)
	}
	if exists {
		t.Error("expected discovery view to not exist")
	}
}

func TestGetDiscoveryView_Found(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(testDiscoveryViewResponse)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	view, err := apiClient.GetDiscoveryView(testDiscoveryViewID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view == nil {
		t.Fatal("expected discovery view, got nil")
	}
	if view.ID != testDiscoveryViewID {
		t.Errorf("unexpected id: %s", view.ID)
	}
	if view.Name != "orca-disco-view-inventory-by-account" {
		t.Errorf("unexpected name: %s", view.Name)
	}
}

func TestGetDiscoveryView_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	view, err := apiClient.GetDiscoveryView("invalid-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view != nil {
		t.Error("expected nil discovery view for 404 response")
	}
}

func TestDeleteDiscoveryView_AlreadyDeleted(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
			Request:    req,
		}
	})}

	apiClient := newTestAPIClient(httpClient)
	if err := apiClient.DeleteDiscoveryView(testDiscoveryViewID); err != nil {
		t.Fatalf("expected no error when deleting an already-deleted view, got: %v", err)
	}
}
