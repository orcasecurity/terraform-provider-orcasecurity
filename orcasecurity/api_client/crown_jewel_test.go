package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const testCrownJewelGroupID = "tf-wasp-1553-probe-do-not-keep"

func TestGetCrownJewel(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "GET", "/api/attack_paths/crown_jewels")
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`[
				{"group_unique_id":"vm_other","description":"keep me"},
				{"group_unique_id":"` + testCrownJewelGroupID + `","description":"tf-probe"}
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

// SetCrownJewel POSTs the reason then GETs for read-your-writes.
func TestSetCrownJewel(t *testing.T) {
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
					{"group_unique_id":"` + testCrownJewelGroupID + `","description":"Customer data"}
				]`)),
				Request: req,
			}
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil
		}
	})}

	client := newTestAPIClient(httpClient)
	jewel, err := client.SetCrownJewel(testCrownJewelGroupID, "Customer data", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jewel == nil || jewel.GroupUniqueID != testCrownJewelGroupID || jewel.Description != "Customer data" {
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
	if payload["description"] != "Customer data" {
		t.Errorf("unexpected description in payload: %v", payload["description"])
	}
}

func TestSetCrownJewel_EmptyDescriptionRejected(t *testing.T) {
	client := newTestAPIClient(&http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		t.Fatal("empty description must not call the API")
		return nil
	})})
	if _, err := client.SetCrownJewel(testCrownJewelGroupID, "  ", 0); err == nil {
		t.Fatal("expected error for empty/whitespace description")
	}
}

func TestSetCrownJewel_RefetchMissSurfacesError(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		switch req.Method {
		case "POST":
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"status":"success"}`)),
				Request:    req,
			}
		case "GET":
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Request:    req,
			}
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil
		}
	})}

	client := newTestAPIClient(httpClient)
	_, err := client.SetCrownJewel(testCrownJewelGroupID, "desc", 0)
	if err == nil {
		t.Fatal("expected error when refetch misses the written jewel")
	}
	if !strings.Contains(err.Error(), testCrownJewelGroupID) {
		t.Errorf("error must name the group_unique_id, got: %v", err)
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
	if err := client.DeleteCrownJewel(testCrownJewelGroupID, 0); err != nil {
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
	if _, ok := payload["description"]; ok {
		t.Errorf("DELETE body must omit description, got %v", payload)
	}
}

func TestInventoryGroupExists(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "POST", "/api/serving-layer/query")
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{
				"status":"success",
				"data":[{
					"group_unique_id":"` + testCrownJewelGroupID + `",
					"data":{"GroupUniqueId":{"value":"` + testCrownJewelGroupID + `"},"DetectedCrownJewelScore":{"value":0},"IsCrownJewel":{"value":false}}
				}]
			}`)),
			Request: req,
		}
	})}
	client := newTestAPIClient(httpClient)
	ok, err := client.InventoryGroupExists(testCrownJewelGroupID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected inventory hit")
	}
}

func TestIsOrcaDetected_Threshold(t *testing.T) {
	cases := []struct {
		score int
		want  bool
	}{
		{0, false},
		{19, false},
		{20, true},
		{75, true},
	}
	for _, tc := range cases {
		g := &InventoryGroup{DetectedCrownJewelScore: tc.score}
		if got := g.IsOrcaDetected(); got != tc.want {
			t.Errorf("score %d: IsOrcaDetected()=%v, want %v", tc.score, got, tc.want)
		}
	}
	if (*InventoryGroup)(nil).IsOrcaDetected() {
		t.Error("nil inventory must not be Orca-detected")
	}
}

func TestGetInventoryGroup_OrcaDetected(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{
				"status":"success",
				"data":[{
					"group_unique_id":"vm_orca",
					"data":{"DetectedCrownJewelScore":{"value":75},"IsCrownJewel":{"value":true}}
				}]
			}`)),
			Request: req,
		}
	})}
	client := newTestAPIClient(httpClient)
	g, err := client.GetInventoryGroup("vm_orca")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g == nil || !g.IsOrcaDetected() || g.DetectedCrownJewelScore != 75 {
		t.Fatalf("expected orca-detected inventory, got %+v", g)
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
	ok, err := client.InventoryGroupExists("tf-phantom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected miss for unknown group_unique_id")
	}
}

func TestCrownJewelWriteClient_DisablesTimeoutRetry(t *testing.T) {
	orig := &APIClient{HTTPClient: &http.Client{Timeout: defaultHTTPTimeout}}
	w := orig.crownJewelWriteClient(0)
	if !w.disableTimeoutRetry {
		t.Fatal("write client must not retry client timeouts")
	}
	if w.HTTPClient.Timeout != DefaultCrownJewelTimeout {
		t.Fatalf("write timeout = %v, want %v", w.HTTPClient.Timeout, DefaultCrownJewelTimeout)
	}
	if orig.disableTimeoutRetry {
		t.Fatal("original client must be unchanged")
	}
	if orig.HTTPClient.Timeout != defaultHTTPTimeout {
		t.Fatal("original HTTP timeout must be unchanged")
	}
}

func TestCrownJewelWriteClient_UsesProvidedTimeout(t *testing.T) {
	orig := &APIClient{HTTPClient: &http.Client{Timeout: defaultHTTPTimeout}}
	custom := 90 * time.Second
	w := orig.crownJewelWriteClient(custom)
	if w.HTTPClient.Timeout != custom {
		t.Fatalf("write timeout = %v, want %v", w.HTTPClient.Timeout, custom)
	}
}
