package servicenow

import (
	"context"
	"net/http"
	"net/http/httptest"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwresourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// newTestClient points a real *api_client.APIClient at an httptest server. This lets the CRUD
// handlers run their full success / not-found / error branches against a controllable HTTP
// backend without touching the live Orca lab (whose ServiceNow create path validates credentials
// and rejects fake ones).
func newTestClient(t *testing.T, handler http.HandlerFunc) *api_client.APIClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	endpoint := srv.URL
	token := "test-token"
	client, err := api_client.NewAPIClient(&endpoint, &token)
	if err != nil {
		t.Fatalf("failed to build api client: %v", err)
	}
	return client
}

// jsonOK returns a handler that writes the given success body as JSON.
func jsonOK(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

// httpError returns a handler that fails every request with the given status.
func httpError(status int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"message":"` + message + `"}`))
	}
}

// itsmUnderTest wires a serviceNowITSMResource to an httptest backend and returns it together
// with the resource schema used to seed tfsdk.State/Plan.
func itsmUnderTest(t *testing.T, handler http.HandlerFunc) (*serviceNowITSMResource, fwresourceschema.Schema) {
	t.Helper()
	return &serviceNowITSMResource{apiClient: newTestClient(t, handler)}, resourceSchema(t)
}

// resourceSchema returns the managed resource's framework schema, used to seed tfsdk.State/Plan.
func resourceSchema(t *testing.T) fwresourceschema.Schema {
	t.Helper()
	r := NewServiceNowResource().(resource.ResourceWithConfigure)
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema build failed: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// stateWith builds a tfsdk.State populated from the given model, using the resource schema.
func stateWith(t *testing.T, sch fwresourceschema.Schema, model serviceNowITSMResourceModel) tfsdk.State {
	t.Helper()
	st := tfsdk.State{Schema: sch}
	if diags := st.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("failed to seed state: %v", diags)
	}
	return st
}

// planWith builds a tfsdk.Plan populated from the given model, using the resource schema.
func planWith(t *testing.T, sch fwresourceschema.Schema, model serviceNowITSMResourceModel) tfsdk.Plan {
	t.Helper()
	p := tfsdk.Plan{Schema: sch}
	if diags := p.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("failed to seed plan: %v", diags)
	}
	return p
}

// minModel returns a fully populated minimal model for tests that only care about the request
// wiring, not the field values.
func minModel(id string) serviceNowITSMResourceModel {
	return serviceNowITSMResourceModel{
		ID:       types.StringValue(id),
		Name:     types.StringValue("n"),
		URL:      types.StringValue("u"),
		Username: types.StringValue("user"),
		Password: types.StringValue("p"),
	}
}

// prodPlanModel returns the plan used by the Create tests: unknown id, real-looking fields.
func prodPlanModel() serviceNowITSMResourceModel {
	return serviceNowITSMResourceModel{
		ID:       types.StringUnknown(),
		Name:     types.StringValue("prod"),
		URL:      types.StringValue("https://acme.service-now.com"),
		Username: types.StringValue("svc-orca"),
		Password: types.StringValue("s3cret"),
	}
}

// Read must populate name/url/username from the API and preserve the password already in state
// (the API strips it). This exercises the happy path.
func TestRead_PopulatesFromAPIAndPreservesPassword(t *testing.T) {
	r, sch := itsmUnderTest(t, jsonOK(`{"status":"success","data":{"id":"res-1","name":"prod","host_url":"https://acme.service-now.com","data":{"username":"svc-orca"}}}`))
	req := resource.ReadRequest{State: stateWith(t, sch, serviceNowITSMResourceModel{
		ID:       types.StringValue("res-1"),
		Name:     types.StringValue("stale"),
		URL:      types.StringValue("stale"),
		Username: types.StringValue("stale"),
		Password: types.StringValue("kept-secret"),
	})}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: sch}}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags: %v", resp.Diagnostics)
	}

	var out serviceNowITSMResourceModel
	if diags := resp.State.Get(context.Background(), &out); diags.HasError() {
		t.Fatalf("failed to read resulting state: %v", diags)
	}
	if out.Name.ValueString() != "prod" {
		t.Errorf("name not refreshed from API: %q", out.Name.ValueString())
	}
	if out.URL.ValueString() != "https://acme.service-now.com" {
		t.Errorf("url not refreshed from API: %q", out.URL.ValueString())
	}
	if out.Username.ValueString() != "svc-orca" {
		t.Errorf("username not refreshed from API: %q", out.Username.ValueString())
	}
	if out.Password.ValueString() != "kept-secret" {
		t.Errorf("password should be preserved from prior state, got %q", out.Password.ValueString())
	}
}

// A 404 from the API must remove the resource from state (no error) so Terraform can plan a
// recreate rather than crash.
func TestRead_NotFoundRemovesResource(t *testing.T) {
	r, sch := itsmUnderTest(t, httpError(http.StatusNotFound, "not found"))
	req := resource.ReadRequest{State: stateWith(t, sch, minModel("gone"))}
	// Seed resp.State with the same non-null raw so we can detect removal (Raw becomes null).
	resp := &resource.ReadResponse{State: stateWith(t, sch, minModel("gone"))}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("404 should not error, got: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("expected resource to be removed from state (null raw), got %v", resp.State.Raw)
	}
}

// A non-404 API failure on Read must surface an error diagnostic.
func TestRead_APIErrorSurfacesDiag(t *testing.T) {
	r, sch := itsmUnderTest(t, httpError(http.StatusInternalServerError, "boom"))
	req := resource.ReadRequest{State: stateWith(t, sch, minModel("x"))}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: sch}}
	r.Read(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic on API failure")
	}
}

// Create must POST the payload and write the API's returned id/name/url/username back to state.
func TestCreate_Success(t *testing.T) {
	r, sch := itsmUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		jsonOK(`{"status":"success","data":{"id":"new-id","name":"prod","host_url":"https://acme.service-now.com","data":{"username":"svc-orca"}}}`)(w, req)
	})
	req := resource.CreateRequest{Plan: planWith(t, sch, prodPlanModel())}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags: %v", resp.Diagnostics)
	}
	var out serviceNowITSMResourceModel
	resp.State.Get(context.Background(), &out)
	if out.ID.ValueString() != "new-id" {
		t.Errorf("id not set from API response: %q", out.ID.ValueString())
	}
	if out.Password.ValueString() != "s3cret" {
		t.Errorf("password should be preserved from plan, got %q", out.Password.ValueString())
	}
}

// A Create API error must surface a diagnostic and not write state.
func TestCreate_APIErrorSurfacesDiag(t *testing.T) {
	r, sch := itsmUnderTest(t, httpError(http.StatusBadRequest, "Invalid credentials or ServiceNow instance unavailable."))
	req := resource.CreateRequest{Plan: planWith(t, sch, prodPlanModel())}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch}}
	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic on Create failure")
	}
}

// Update must PUT the payload and refresh state from the API response.
func TestUpdate_Success(t *testing.T) {
	r, sch := itsmUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", req.Method)
		}
		jsonOK(`{"status":"success","data":{"id":"res-1","name":"renamed","host_url":"https://acme.service-now.com","data":{"username":"svc-orca"}}}`)(w, req)
	})
	renamed := prodPlanModel()
	renamed.ID = types.StringValue("res-1")
	renamed.Name = types.StringValue("renamed")
	prior := prodPlanModel()
	prior.ID = types.StringValue("res-1")
	req := resource.UpdateRequest{
		Plan:  planWith(t, sch, renamed),
		State: stateWith(t, sch, prior),
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: sch}}
	r.Update(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags: %v", resp.Diagnostics)
	}
	var out serviceNowITSMResourceModel
	resp.State.Get(context.Background(), &out)
	if out.Name.ValueString() != "renamed" {
		t.Errorf("name not refreshed after update: %q", out.Name.ValueString())
	}
}

// An Update API error must surface a diagnostic.
func TestUpdate_APIErrorSurfacesDiag(t *testing.T) {
	r, sch := itsmUnderTest(t, httpError(http.StatusInternalServerError, "boom"))
	req := resource.UpdateRequest{
		Plan:  planWith(t, sch, minModel("res-1")),
		State: stateWith(t, sch, minModel("res-1")),
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: sch}}
	r.Update(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic on Update failure")
	}
}

// Delete must issue the DELETE and complete without diagnostics on success.
func TestDelete_Success(t *testing.T) {
	r, sch := itsmUnderTest(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		jsonOK(`{"status":"success"}`)(w, req)
	})
	req := resource.DeleteRequest{State: stateWith(t, sch, minModel("res-1"))}
	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: sch}}
	r.Delete(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags on delete: %v", resp.Diagnostics)
	}
}

// A Delete API error must surface a diagnostic.
func TestDelete_APIErrorSurfacesDiag(t *testing.T) {
	r, sch := itsmUnderTest(t, httpError(http.StatusInternalServerError, "boom"))
	req := resource.DeleteRequest{State: stateWith(t, sch, minModel("res-1"))}
	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: sch}}
	r.Delete(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic on Delete failure")
	}
}

// ImportState must copy the import ID into the id attribute.
func TestImportState_SetsID(t *testing.T) {
	sch := resourceSchema(t)
	r := &serviceNowITSMResource{}
	resp := &resource.ImportStateResponse{State: tfsdk.State{
		Schema: sch,
		// Import starts from an all-null object of the schema type.
		Raw: tftypes.NewValue(sch.Type().TerraformType(context.Background()), nil),
	}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "imported-id"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diags on import: %v", resp.Diagnostics)
	}
	var out serviceNowITSMResourceModel
	resp.State.Get(context.Background(), &out)
	if out.ID.ValueString() != "imported-id" {
		t.Errorf("import did not set id, got %q", out.ID.ValueString())
	}
}
