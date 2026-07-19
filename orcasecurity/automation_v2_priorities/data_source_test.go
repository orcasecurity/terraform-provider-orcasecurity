package automation_v2_priorities

import (
	"context"
	"io"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// roundTripFunc adapts a function into an http.RoundTripper. api_client.RoundTripFunc
// is defined in api_client's own _test.go file, so it is not visible outside that
// package's test binary; this is a local equivalent for use here.
type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestSchemaHasComputedAutomationsList(t *testing.T) {
	ds := &automationPrioritiesDataSource{}
	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if _, ok := resp.Schema.Attributes["automations"]; !ok {
		t.Fatal("schema must expose an 'automations' attribute")
	}
}

func TestFetchAutomationsMapsServerOrder(t *testing.T) {
	body := `{"total_items": 2, "data": [
		{"id":"a1","name":"first","status":"enabled","filter":{"sonar_query":{"models":["Alert"],"type":"object_set"}},"actions":[],"priority":1},
		{"id":"a2","name":"second","status":"disabled","filter":{"sonar_query":{"models":["Alert"],"type":"object_set"}},"actions":[],"priority":2}
	]}`
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Request: req}
	})}
	ds := &automationPrioritiesDataSource{apiClient: &api_client.APIClient{
		APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient,
	}}

	entries, err := ds.fetchAutomations()
	if err != nil {
		t.Fatalf("fetchAutomations failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ID.ValueString() != "a1" || entries[0].Priority.ValueInt64() != 1 {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Name.ValueString() != "second" || entries[1].Status.ValueString() != "disabled" {
		t.Errorf("unexpected second entry: %+v", entries[1])
	}
}
