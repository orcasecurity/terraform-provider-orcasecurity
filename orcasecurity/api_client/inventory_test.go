package api_client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetInventoryGroup(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "POST", "/api/serving-layer/query")
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{
				"status":"success",
				"data":[{
					"group_unique_id":"` + testCrownJewelGroupID + `",
					"data":{"GroupUniqueId":{"value":"` + testCrownJewelGroupID + `"}}
				}]
			}`)),
			Request: req,
		}
	})}
	client := newTestAPIClient(httpClient)
	g, err := client.GetInventoryGroup(testCrownJewelGroupID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g == nil {
		t.Fatal("expected inventory hit")
	}
}

func TestGetInventoryGroup_Missing(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":[]}`)),
			Request:    req,
		}
	})}
	client := newTestAPIClient(httpClient)
	g, err := client.GetInventoryGroup("tf-phantom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g != nil {
		t.Fatalf("expected miss for unknown group_unique_id, got %+v", g)
	}
}
