package automation_v2_priorities

import (
	"context"
	"io"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// stubDataSource returns a data source whose API always answers with body.
func stubDataSource(body string) *automationPrioritiesDataSource {
	return &automationPrioritiesDataSource{apiClient: testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Request: req}
	})}
}

func TestSchemaHasComputedAutomationsList(t *testing.T) {
	ds := &automationPrioritiesDataSource{}
	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	attr, ok := resp.Schema.Attributes["automations"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("schema must expose 'automations' as a ListNestedAttribute")
	}
	if !attr.Computed {
		t.Error("automations must be Computed")
	}
}

func TestFetchAutomationsMapsServerOrder(t *testing.T) {
	ds := stubDataSource(`{"total_items": 2, "data": [
		{"id":"a1","name":"first","status":"enabled","filter":{"sonar_query":{"models":["Alert"],"type":"object_set"}},"actions":[],"priority":1},
		{"id":"a2","name":"second","status":"disabled","filter":{"sonar_query":{"models":["Alert"],"type":"object_set"}},"actions":[],"priority":2}
	]}`)

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

func TestFetchAutomationsEmpty(t *testing.T) {
	ds := stubDataSource(`{"total_items": 0, "data": []}`)

	entries, err := ds.fetchAutomations()
	if err != nil {
		t.Fatalf("fetchAutomations failed: %v", err)
	}
	if entries == nil {
		t.Fatal("expected empty (non-nil) slice for zero automations")
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}
