package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestInventoryGroupExists(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "POST", "/api/serving-layer/query")
		var body servingLayerQueryRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(body.Select) != 1 || body.Select[0] != "GroupUniqueId" {
			t.Errorf("Select must project GroupUniqueId, got %v", body.Select)
		}
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{
				"status":"success",
				"data":[{
					"data":{"GroupUniqueId":{"value":"` + testCrownJewelGroupID + `"}}
				}]
			}`)),
			Request: req,
		}
	})}
	client := newTestAPIClient(httpClient)
	exists, err := client.InventoryGroupExists(testCrownJewelGroupID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected inventory hit")
	}
}

func TestInventoryGroupExists_Missing(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":[]}`)),
			Request:    req,
		}
	})}
	client := newTestAPIClient(httpClient)
	exists, err := client.InventoryGroupExists("tf-phantom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected miss for unknown group_unique_id")
	}
}
