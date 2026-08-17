package shift_left_policy

// Delete is exercised directly rather than through resource.Test: the unrecoverable case leaves a
// policy the API will not remove, which the acceptance harness cannot tear down.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const lastActiveBody = `{"message":"Policy cannot be removed/disabled since it's the last active policy of the following projects: ['proj-1(1111)']"}`

type deleteStub struct {
	detachStatus int
	deletes      int
	detaches     int
	detached     bool
}

func (s *deleteStub) client(t *testing.T) *api_client.APIClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/projects/"):
			s.detaches++
			if s.detachStatus != http.StatusOK {
				http.Error(w, lastActiveBody, s.detachStatus)
				return
			}
			s.detached = true
			_, _ = w.Write([]byte(`{"id":"policy-1","projects":[]}`))
		case r.Method == http.MethodDelete:
			s.deletes++
			if !s.detached {
				http.Error(w, lastActiveBody, http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	return &api_client.APIClient{APIEndpoint: srv.URL, APIToken: "stub", HTTPClient: srv.Client()}
}

func deletePolicy(t *testing.T, stub *deleteStub) *resource.DeleteResponse {
	t.Helper()
	r := &shiftLeftPolicyResource{apiClient: stub.client(t)}
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, &shiftLeftPolicyResourceModel{
		ID:                       types.StringValue("policy-1"),
		Type:                     types.StringValue("malicious_packages"),
		Name:                     types.StringValue("mp"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		ProjectsIds:              types.ListNull(types.StringType),
		AttachAllProjects:        types.BoolValue(true),
		Builtin:                  types.BoolValue(false),
	}); diags.HasError() {
		t.Fatalf("failed to seed state: %v", diags)
	}

	resp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
	return resp
}

func TestDelete_LastActivePolicySucceedsWhenDetachIsAllowed(t *testing.T) {
	stub := &deleteStub{detachStatus: http.StatusOK}
	resp := deletePolicy(t, stub)

	if resp.Diagnostics.HasError() {
		t.Fatalf("delete should recover via a detach the API allows: %v", resp.Diagnostics)
	}
	if stub.detaches != 1 || stub.deletes != 2 {
		t.Errorf("expected 1 detach and a retried delete, got %d detaches / %d deletes", stub.detaches, stub.deletes)
	}
}

func TestDelete_LastActivePolicyReportsActionableErrorWhenDetachIsRefused(t *testing.T) {
	stub := &deleteStub{detachStatus: http.StatusBadRequest}
	resp := deletePolicy(t, stub)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when both the delete and the detach are refused")
	}
	summary := resp.Diagnostics.Errors()[0].Summary()
	if !strings.Contains(summary, "last active policy") {
		t.Errorf("unexpected summary: %s", summary)
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	// The operator has to resolve this outside Terraform, so the reason and the type must be named.
	if !strings.Contains(detail, "malicious_packages") || !strings.Contains(detail, "proj-1") {
		t.Errorf("detail must name the policy type and the blocking projects: %s", detail)
	}
	if stub.deletes != 1 {
		t.Errorf("a refused detach must not trigger a delete retry, got %d deletes", stub.deletes)
	}
}
