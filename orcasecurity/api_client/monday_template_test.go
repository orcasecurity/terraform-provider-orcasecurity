package api_client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"
)

func newMondayTestClient(handler func(req *http.Request) *http.Response) *api_client.APIClient {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(handler)}
	return &api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
}

func TestMondayTemplate_CreateSendsServiceNameResourceAndConfig(t *testing.T) {
	var gotBody map[string]interface{}
	apiClient := newMondayTestClient(func(req *http.Request) *http.Response {
		raw, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(raw, &gotBody)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"id":"cfg-1","service_name":"monday","template_name":"t1","resource":"res-1","is_enabled":true,"is_default":false,"config":{"board_id":"b1"}}}`)),
			Request:    req,
		}
	})

	out, err := apiClient.CreateMondayTemplate(api_client.MondayTemplate{
		TemplateName: "t1",
		Resource:     "res-1",
		IsEnabled:    true,
		Config: api_client.MondayTemplateConfig{
			BoardID:     "b1",
			WorkspaceID: "w1",
			Mapping:     json.RawMessage(`{"status_14":{"value":"0"}}`),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != "cfg-1" {
		t.Errorf("expected id cfg-1, got %q", out.ID)
	}
	if gotBody["service_name"] != "monday" {
		t.Errorf("expected service_name monday, got %v", gotBody["service_name"])
	}
	if gotBody["resource"] != "res-1" {
		t.Errorf("expected top-level resource res-1, got %v", gotBody["resource"])
	}
	cfg := gotBody["config"].(map[string]interface{})
	if cfg["board_id"] != "b1" || cfg["workspace_id"] != "w1" {
		t.Errorf("unexpected config: %v", cfg)
	}
}

func TestMondayTemplate_UpdateOmitsBusinessUnitsAndKeepsResource(t *testing.T) {
	var gotBody map[string]interface{}
	apiClient := newMondayTestClient(func(req *http.Request) *http.Response {
		raw, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(raw, &gotBody)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"id":"cfg-1","resource":"res-1","config":{"board_id":"b1"}}}`)),
			Request:    req,
		}
	})

	_, err := apiClient.UpdateMondayTemplate("t1", api_client.MondayTemplate{
		Resource:      "res-1",
		IsEnabled:     true,
		IsDefault:     false,
		BusinessUnits: []string{"bu-1"},
		Config:        api_client.MondayTemplateConfig{BoardID: "b1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := gotBody["business_units"]; present {
		t.Errorf("business_units must be omitted from PUT body, got %v", gotBody["business_units"])
	}
	if gotBody["resource"] != "res-1" {
		t.Errorf("expected resource res-1 in PUT body, got %v", gotBody["resource"])
	}
}

func TestMondayTemplate_GetReturnsFirstEntry(t *testing.T) {
	apiClient := newMondayTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":[{"id":"cfg-1","template_name":"t1","resource":"res-1","config":{"board_id":"b1","group_id":"topics"}}]}`)),
			Request:    req,
		}
	})

	out, err := apiClient.GetMondayTemplate("t1")
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || out.ID != "cfg-1" {
		t.Fatalf("expected cfg-1, got %+v", out)
	}
	if out.Config.BoardID != "b1" || out.Config.GroupID != "topics" {
		t.Errorf("unexpected config: %+v", out.Config)
	}
	if out.Resource != "res-1" {
		t.Errorf("expected top-level resource res-1 to round-trip, got %q", out.Resource)
	}
}
