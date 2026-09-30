package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const businessUnitResponse = `{"status":"success","data":{
	"id":"8f0c5e1a-1111-4222-8333-444455556666","name":"bu","bu_type":"combined_filter",
	"config":{"some":{"CloudProviders":["aws"]}},"global_filter":true,
	"filter_data":{"cloud_provider":["aws"]},"is_populating":false,
	"business_criticality":"","owner_team":"","application":"","contact_emails":[],"deployment_stages":[]}}`

func businessUnitClient(t *testing.T, handler func(req *http.Request) *http.Response) APIClient {
	t.Helper()
	return APIClient{
		APIEndpoint: "http://localhost",
		APIToken:    "secret",
		HTTPClient:  &http.Client{Transport: RoundTripFunc(handler)},
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func TestGetBusinessUnit(t *testing.T) {
	client := businessUnitClient(t, func(req *http.Request) *http.Response {
		if req.Method != "GET" || req.URL.Path != "/api/business_units/8f0c5e1a-1111-4222-8333-444455556666" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		return jsonResponse(200, businessUnitResponse)
	})

	bu, err := client.GetBusinessUnit("8f0c5e1a-1111-4222-8333-444455556666")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bu.ID != "8f0c5e1a-1111-4222-8333-444455556666" || bu.Name != "bu" {
		t.Errorf("unexpected business unit: %+v", bu)
	}
	if string(bu.Config) != `{"some":{"CloudProviders":["aws"]}}` {
		t.Errorf("unexpected config: %s", bu.Config)
	}
	if bu.GlobalFilter == nil || !*bu.GlobalFilter {
		t.Errorf("expected global_filter true, got %v", bu.GlobalFilter)
	}
}

func TestGetBusinessUnit_NotFound(t *testing.T) {
	client := businessUnitClient(t, func(req *http.Request) *http.Response {
		return jsonResponse(404, `{"status":"failure","error":"Not found."}`)
	})

	bu, err := client.GetBusinessUnit("missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bu != nil {
		t.Errorf("expected nil for missing business unit, got %+v", bu)
	}
}

func TestGetBusinessUnit_ServerError(t *testing.T) {
	client := businessUnitClient(t, func(req *http.Request) *http.Response {
		return jsonResponse(403, `{"status":"failure","error":"Forbidden"}`)
	})

	if _, err := client.GetBusinessUnit("id"); err == nil {
		t.Error("expected an error for a non-404 failure")
	}
}

func TestCreateBusinessUnit(t *testing.T) {
	client := businessUnitClient(t, func(req *http.Request) *http.Response {
		if req.Method != "POST" || req.URL.Path != "/api/business_units" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		body, _ := io.ReadAll(req.Body)
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("payload not JSON: %v", err)
		}
		if string(payload["bu_type"]) != `"combined_filter"` {
			t.Errorf("unexpected bu_type: %s", payload["bu_type"])
		}
		if string(payload["config"]) != `{"some":{"CloudProviders":["aws"]}}` {
			t.Errorf("unexpected config: %s", payload["config"])
		}
		for _, legacy := range []string{"filter_data", "shiftleft_filter_data", "filter_id"} {
			if _, found := payload[legacy]; found {
				t.Errorf("payload must not carry legacy field %s: %s", legacy, body)
			}
		}
		return jsonResponse(201, businessUnitResponse)
	})

	bu, err := client.CreateBusinessUnit(BusinessUnit{
		Name:   "bu",
		BUType: BusinessUnitTypeCombinedFilter,
		Config: json.RawMessage(`{"some":{"CloudProviders":["aws"]}}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bu.ID != "8f0c5e1a-1111-4222-8333-444455556666" {
		t.Errorf("unexpected id: %s", bu.ID)
	}
}

func TestUpdateBusinessUnit(t *testing.T) {
	client := businessUnitClient(t, func(req *http.Request) *http.Response {
		if req.Method != "PUT" || req.URL.Path != "/api/business_units/8f0c5e1a-1111-4222-8333-444455556666" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		return jsonResponse(200, businessUnitResponse)
	})

	bu, err := client.UpdateBusinessUnit("8f0c5e1a-1111-4222-8333-444455556666", BusinessUnit{
		Name:   "bu",
		BUType: BusinessUnitTypeCombinedFilter,
		Config: json.RawMessage(`{"some":{"CloudProviders":["aws"]}}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bu.Name != "bu" {
		t.Errorf("unexpected name: %s", bu.Name)
	}
}

func TestDeleteBusinessUnit(t *testing.T) {
	for name, status := range map[string]int{"deleted": 204, "already gone": 404} {
		t.Run(name, func(t *testing.T) {
			client := businessUnitClient(t, func(req *http.Request) *http.Response {
				if req.Method != "DELETE" || req.URL.Path != "/api/business_units/id" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				return jsonResponse(status, "")
			})
			if err := client.DeleteBusinessUnit("id"); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
