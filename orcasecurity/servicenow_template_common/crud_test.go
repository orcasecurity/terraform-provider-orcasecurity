package servicenow_template_common

import (
	"context"
	"net/http"
	"net/http/httptest"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwresourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// itsmOptions builds the ITSM variant Options used to construct a resource in tests.
func itsmOptions() Options {
	return Options{
		TypeNameSuffix: "_integration_servicenow_itsm_template",
		UIName:         "ServiceNow ITSM template",
		ConfigType:     api_client.ServiceNowITSMTemplateConfigType,
		Create:         (*api_client.APIClient).CreateServiceNowITSMTemplate,
		Get:            (*api_client.APIClient).GetServiceNowITSMTemplate,
		Update:         (*api_client.APIClient).UpdateServiceNowITSMTemplate,
		Delete:         (*api_client.APIClient).DeleteServiceNowITSMTemplate,
	}
}

// configuredResource returns a resource wired to an httptest-backed API client. Driving the
// resource through its real Create/Read exercises the BuildPayload and Extract closures inside
// NewResource — the only place that logic runs — without hitting the live Orca lab (which rejects
// unvalidated ServiceNow credentials).
func configuredResource(t *testing.T, opts Options, handler http.HandlerFunc) (resource.Resource, fwresourceschema.Schema, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	endpoint := srv.URL
	token := "test-token"
	client, err := api_client.NewAPIClient(&endpoint, &token)
	if err != nil {
		srv.Close()
		t.Fatalf("failed to build api client: %v", err)
	}
	r := NewResource(opts)
	r.(resource.ResourceWithConfigure).Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, &resource.ConfigureResponse{})

	sresp := &resource.SchemaResponse{}
	r.(resource.ResourceWithConfigure).Schema(context.Background(), resource.SchemaRequest{}, sresp)
	if sresp.Diagnostics.HasError() {
		srv.Close()
		t.Fatalf("schema build failed: %v", sresp.Diagnostics)
	}
	return r, sresp.Schema, srv.Close
}

// planModel seeds a tfsdk.Plan from a state model using the resource schema.
func planModel(t *testing.T, sch fwresourceschema.Schema, model *state) tfsdk.Plan {
	t.Helper()
	p := tfsdk.Plan{Schema: sch}
	if diags := p.Set(context.Background(), model); diags.HasError() {
		t.Fatalf("failed to seed plan: %v", diags)
	}
	return p
}

// stateModel seeds a tfsdk.State from a state model using the resource schema.
func stateModel(t *testing.T, sch fwresourceschema.Schema, model *state) tfsdk.State {
	t.Helper()
	st := tfsdk.State{Schema: sch}
	if diags := st.Set(context.Background(), model); diags.HasError() {
		t.Fatalf("failed to seed state: %v", diags)
	}
	return st
}

// newModel returns a fully-specified state model. Every Optional attribute must be set (not left
// as the Go zero value) because the framework requires plan values to be known.
func newModel() *state {
	m := &state{}
	m.TemplateName = types.StringValue("tf-tmpl")
	m.IsEnabled = types.BoolValue(true)
	m.IsDefault = types.BoolValue(false)
	m.ResourceID = types.StringValue("res-1")
	m.InstanceName = types.StringValue("acme")
	m.BaseURL = types.StringNull()
	m.Username = types.StringValue("svc-orca")
	m.ResolutionStatus = types.StringValue("6")
	m.ResolutionCode = types.StringValue("Solved")
	m.ResolutionNote = types.StringValue("done")
	m.ReopenStatus = types.StringValue("2")
	m.MappingJSON = common.NewOrcaMappingValue(`{"short_description":["alert_id"]}`)
	m.OnCloseAlertMappingJSON = jsontypes.NewNormalizedValue(`{"state":"closed"}`)
	m.AllowReopenAndResolution = types.BoolValue(true)
	m.AllowMapping = types.BoolValue(true)
	return m
}

// Create must run the BuildPayload closure (encoding mapping_json into the wire form + pinning
// config.type) and then apply the API response through Extract back into state.
func TestResourceCreate_BuildPayloadAndExtract(t *testing.T) {
	var receivedBody string
	r, sch, closeFn := configuredResource(t, itsmOptions(), func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			buf := make([]byte, req.ContentLength)
			req.Body.Read(buf)
			receivedBody = string(buf)
		}
		w.Header().Set("content-type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"id":"tmpl-id","template_name":"tf-tmpl","is_enabled":true,"is_default":false,"resource":"res-1","config":{"type":"ITSM","instance_name":"acme","username":"svc-orca","resolution_status":"6","mapping":{"short_description":[{"orca":"alert_id"}]},"allow_mapping":true,"allow_reopen_and_resolution":true}}}`))
	})
	defer closeFn()

	req := resource.CreateRequest{Plan: planModel(t, sch, newModel())}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags: %v", resp.Diagnostics)
	}

	// BuildPayload must have expanded the shorthand mapping into the {"orca":...} wire form and
	// pinned config.type = ITSM.
	if !contains(receivedBody, `"orca":"alert_id"`) {
		t.Errorf("payload did not expand mapping shorthand to wire form: %s", receivedBody)
	}
	if !contains(receivedBody, `"type":"ITSM"`) {
		t.Errorf("payload did not pin config.type=ITSM: %s", receivedBody)
	}

	var out state
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatalf("failed to read resulting state: %v", diags)
	}
	if out.ID.ValueString() != "tmpl-id" {
		t.Errorf("id not set from API: %q", out.ID.ValueString())
	}
	if out.InstanceName.ValueString() != "acme" {
		t.Errorf("instance_name not extracted: %q", out.InstanceName.ValueString())
	}
}

// Read must fetch the template, run Extract (populating variant fields) and write state.
func TestResourceRead_ExtractPopulatesState(t *testing.T) {
	r, sch, closeFn := configuredResource(t, itsmOptions(), func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("content-type", "application/json")
		// GET returns a list envelope filtered by config.type == ITSM.
		w.Write([]byte(`{"status":"success","data":[{"id":"tmpl-id","template_name":"tf-tmpl","is_enabled":true,"is_default":false,"resource":"res-9","config":{"type":"ITSM","instance_name":"beta","username":"u2","resolution_status":"7"}}]}`))
	})
	defer closeFn()

	req := resource.ReadRequest{State: stateModel(t, sch, newModel())}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: sch}}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags: %v", resp.Diagnostics)
	}
	var out state
	resp.State.Get(context.Background(), &out)
	if out.ResourceID.ValueString() != "res-9" {
		t.Errorf("resource_id not refreshed from API: %q", out.ResourceID.ValueString())
	}
	if out.InstanceName.ValueString() != "beta" {
		t.Errorf("instance_name not refreshed from API: %q", out.InstanceName.ValueString())
	}
}

// Read must remove the resource from state when the API returns no matching template.
func TestResourceRead_NoMatchRemovesResource(t *testing.T) {
	r, sch, closeFn := configuredResource(t, itsmOptions(), func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.Write([]byte(`{"status":"success","data":[]}`))
	})
	defer closeFn()

	req := resource.ReadRequest{State: stateModel(t, sch, newModel())}
	resp := &resource.ReadResponse{State: stateModel(t, sch, newModel())}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("expected resource removed from state (null raw)")
	}
}

// Delete must succeed without diagnostics.
func TestResourceDelete_Success(t *testing.T) {
	r, sch, closeFn := configuredResource(t, itsmOptions(), func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.Write([]byte(`{"status":"success"}`))
	})
	defer closeFn()

	req := resource.DeleteRequest{State: stateModel(t, sch, newModel())}
	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: sch}}
	r.Delete(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags on delete: %v", resp.Diagnostics)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
