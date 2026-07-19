package monday_resource

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// buildPayload must place the plan's name at the top level and route the api_token into the
// nested Data block as a pointer (the field is write-only / omitempty on the wire).
func TestBuildPayload_PopulatesNameAndToken(t *testing.T) {
	r := &mondayResource{}
	plan := mondayResourceModel{
		Name:     types.StringValue("my-monday"),
		APIToken: types.StringValue("secret-token"),
	}

	payload := r.buildPayload(plan)

	if payload.Name != "my-monday" {
		t.Errorf("name mismatch: got %q", payload.Name)
	}
	if payload.Data.APIToken == nil {
		t.Fatal("expected api_token pointer to be set, got nil")
	}
	if *payload.Data.APIToken != "secret-token" {
		t.Errorf("api_token mismatch: got %q", *payload.Data.APIToken)
	}
}

// An empty api_token still produces a non-nil pointer to "" — buildPayload does not decide
// omission; the schema's LengthAtLeast(1) validator guards against empty tokens upstream.
func TestBuildPayload_EmptyTokenIsNonNilPointer(t *testing.T) {
	r := &mondayResource{}
	plan := mondayResourceModel{
		Name:     types.StringValue("n"),
		APIToken: types.StringValue(""),
	}

	payload := r.buildPayload(plan)

	if payload.Data.APIToken == nil {
		t.Fatal("expected non-nil pointer even for empty token")
	}
	if *payload.Data.APIToken != "" {
		t.Errorf("expected empty token string, got %q", *payload.Data.APIToken)
	}
}

// applyResponse must set the computed ID and account_slug from the API, and refresh the name
// when the API returns one.
func TestApplyResponse_CopiesComputedFields(t *testing.T) {
	m := &mondayResourceModel{
		Name:     types.StringValue("planned-name"),
		APIToken: types.StringValue("keep-me"),
	}
	api := &api_client.MondayResource{
		ID:   "uuid-123",
		Name: "server-name",
		Data: api_client.MondayResourceData{AccountSlug: "orca999"},
	}

	applyResponse(m, api)

	if m.ID.ValueString() != "uuid-123" {
		t.Errorf("id mismatch: got %q", m.ID.ValueString())
	}
	if m.Name.ValueString() != "server-name" {
		t.Errorf("name should be refreshed from API: got %q", m.Name.ValueString())
	}
	if m.AccountSlug.ValueString() != "orca999" {
		t.Errorf("account_slug mismatch: got %q", m.AccountSlug.ValueString())
	}
	// api_token is write-only and never touched by applyResponse — it must survive verbatim.
	if m.APIToken.ValueString() != "keep-me" {
		t.Errorf("api_token must be preserved, got %q", m.APIToken.ValueString())
	}
}

// When the API omits the name (empty string), applyResponse must NOT clobber the planned name
// with "" — it only overwrites when the API actually returns a value.
func TestApplyResponse_EmptyAPINameKeepsPlanned(t *testing.T) {
	m := &mondayResourceModel{
		Name: types.StringValue("planned-name"),
	}
	api := &api_client.MondayResource{
		ID:   "uuid-123",
		Name: "", // API omitted the name
		Data: api_client.MondayResourceData{AccountSlug: "orca999"},
	}

	applyResponse(m, api)

	if m.Name.ValueString() != "planned-name" {
		t.Errorf("planned name should be preserved when API omits name, got %q", m.Name.ValueString())
	}
	if m.ID.ValueString() != "uuid-123" {
		t.Errorf("id should still be set, got %q", m.ID.ValueString())
	}
}

// account_slug is always set from the API, even to empty — an absent slug becomes a known ""
// rather than staying unknown, so the plan can settle.
func TestApplyResponse_EmptySlugStillSet(t *testing.T) {
	m := &mondayResourceModel{}
	api := &api_client.MondayResource{
		ID:   "uuid-123",
		Data: api_client.MondayResourceData{AccountSlug: ""},
	}

	applyResponse(m, api)

	if m.AccountSlug.IsNull() || m.AccountSlug.IsUnknown() {
		t.Errorf("account_slug should be a known value, got %v", m.AccountSlug)
	}
	if m.AccountSlug.ValueString() != "" {
		t.Errorf("account_slug should be empty string, got %q", m.AccountSlug.ValueString())
	}
}
