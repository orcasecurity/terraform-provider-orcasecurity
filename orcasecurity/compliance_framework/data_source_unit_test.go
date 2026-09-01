package compliance_framework

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSingleDataSource_NotFound(t *testing.T) {
	d := &complianceFrameworkDataSource{apiClient: testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"error":"Framework missing not found."}`)),
			Request:    req,
		}
	})}
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)
	cfgModel := singleDataSourceModel{frameworkModel: frameworkModel{
		ID:                    types.StringValue("missing"),
		SelectionScopes:       types.ListNull(types.StringType),
		FrameworkCloudVendors: types.ListNull(types.StringType),
	}, Sections: types.ListNull(catalogSectionObjectType(maxCatalogDepth - 1))}
	st := tfsdk.State{Schema: schemaResp.Schema}
	if diags := st.Set(context.Background(), &cfgModel); diags.HasError() {
		t.Fatalf("set: %v", diags)
	}
	cfg := tfsdk.Config{Schema: schemaResp.Schema, Raw: st.Raw}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(context.Background(), datasource.ReadRequest{Config: cfg}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("missing framework must be a diagnostic, not empty state")
	}
	found := false
	for _, diag := range resp.Diagnostics {
		if strings.Contains(diag.Detail(), "missing") || strings.Contains(diag.Summary(), "not found") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected not-found diagnostic, got %v", resp.Diagnostics)
	}
}
