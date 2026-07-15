package user_access

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nullListsRef returns a reference model whose scope lists are all null, as when the config left
// every optional list unset.
func nullListsRef() *userAccessResourceModel {
	return &userAccessResourceModel{
		CloudAccounts:     types.ListNull(types.StringType),
		ShiftleftProjects: types.ListNull(types.StringType),
		UserFilters:       types.ListNull(types.StringType),
	}
}

// modelToAPI must copy scalar fields verbatim and flatten every scoped list.
func TestModelToAPI_MapsAllFields(t *testing.T) {
	r := &userAccessResource{}
	plan := userAccessResourceModel{
		UserID:            types.StringValue("user-1"),
		RoleID:            types.StringValue("role-1"),
		AllCloudAccounts:  types.BoolValue(true),
		CloudAccounts:     testutils.StringList(t, "ca-1", "ca-2"),
		ShiftleftProjects: testutils.StringList(t, "sl-1"),
		UserFilters:       testutils.StringList(t, "bu-1", "bu-2"),
	}

	got, diags := r.modelToAPI(context.Background(), plan, "assign-99")
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.ID != "assign-99" {
		t.Errorf("id: want assign-99, got %q", got.ID)
	}
	if got.UserID != "user-1" || got.RoleID != "role-1" {
		t.Errorf("scalar mismatch: %+v", got)
	}
	if !got.AllCloudAccounts {
		t.Error("all_cloud_accounts should be true")
	}
	if !testutils.SameElements(got.CloudAccounts, []string{"ca-1", "ca-2"}) {
		t.Errorf("cloud accounts mismatch: %v", got.CloudAccounts)
	}
	if !testutils.SameElements(got.ShiftleftProjects, []string{"sl-1"}) {
		t.Errorf("shiftleft mismatch: %v", got.ShiftleftProjects)
	}
	if !testutils.SameElements(got.UserFilters, []string{"bu-1", "bu-2"}) {
		t.Errorf("user filters mismatch: %v", got.UserFilters)
	}
}

// A null optional list must become a non-nil empty slice so the API receives an
// explicit empty array rather than a dropped field (see StringSliceFromList contract).
func TestModelToAPI_NullListsBecomeEmptySlices(t *testing.T) {
	r := &userAccessResource{}
	plan := *nullListsRef()
	plan.UserID = types.StringValue("user-1")
	plan.RoleID = types.StringValue("role-1")
	plan.AllCloudAccounts = types.BoolValue(false)

	got, diags := r.modelToAPI(context.Background(), plan, "")
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	for name, s := range map[string][]string{
		"cloud_accounts":     got.CloudAccounts,
		"shiftleft_projects": got.ShiftleftProjects,
		"user_filters":       got.UserFilters,
	} {
		if s == nil {
			t.Errorf("%s should be non-nil empty slice, got nil", name)
		}
		if len(s) != 0 {
			t.Errorf("%s should be empty, got %v", name, s)
		}
	}
	if got.ID != "" {
		t.Errorf("empty assignmentID should stay empty, got %q", got.ID)
	}
}

// apiToModel must copy the API scalars into state and echo non-empty scope lists.
func TestApiToModel_PopulatesFromAPI(t *testing.T) {
	r := &userAccessResource{}
	ref := &userAccessResourceModel{
		CloudAccounts:     testutils.StringList(t, "ca-1"),
		ShiftleftProjects: testutils.StringList(t, "sl-1"),
		UserFilters:       testutils.StringList(t, "bu-1"),
	}
	ua := &api_client.UserAccess{
		ID:                "assign-1",
		UserID:            "user-1",
		RoleID:            "role-1",
		AllCloudAccounts:  true,
		CloudAccounts:     []string{"ca-1", "ca-2"},
		ShiftleftProjects: []string{"sl-1"},
		UserFilters:       []string{"bu-1"},
	}

	got, diags := r.apiToModel(context.Background(), ua, ref)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.ID.ValueString() != "assign-1" || got.UserID.ValueString() != "user-1" || got.RoleID.ValueString() != "role-1" {
		t.Errorf("scalar mismatch: %+v", got)
	}
	if !got.AllCloudAccounts.ValueBool() {
		t.Error("all_cloud_accounts should be true")
	}
	var cloud []string
	if d := got.CloudAccounts.ElementsAs(context.Background(), &cloud, false); d.HasError() {
		t.Fatalf("elements as: %v", d)
	}
	if !testutils.SameElements(cloud, []string{"ca-1", "ca-2"}) {
		t.Errorf("cloud accounts mismatch: %v", cloud)
	}
}

// When config left an optional list null and the API returns nothing, apiToModel
// must keep the list null (not []) to avoid a perpetual "null vs []" plan diff.
func TestApiToModel_NullRefStaysNullOnEmptyAPI(t *testing.T) {
	r := &userAccessResource{}
	ua := &api_client.UserAccess{
		ID:               "assign-1",
		UserID:           "user-1",
		RoleID:           "role-1",
		AllCloudAccounts: false,
		// empty scope
		CloudAccounts:     []string{},
		ShiftleftProjects: nil,
		UserFilters:       []string{},
	}

	got, diags := r.apiToModel(context.Background(), ua, nullListsRef())
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !got.CloudAccounts.IsNull() {
		t.Errorf("cloud_accounts should stay null, got %v", got.CloudAccounts)
	}
	if !got.ShiftleftProjects.IsNull() {
		t.Errorf("shiftleft_projects should stay null, got %v", got.ShiftleftProjects)
	}
	if !got.UserFilters.IsNull() {
		t.Errorf("user_filters should stay null, got %v", got.UserFilters)
	}
}

// When config supplied an explicit empty list, an empty API response must round-trip
// to an empty (non-null) list, preserving the user's explicit "[]".
func TestApiToModel_ExplicitEmptyRefStaysEmpty(t *testing.T) {
	r := &userAccessResource{}
	ref := &userAccessResourceModel{
		CloudAccounts:     testutils.StringList(t), // explicit []
		ShiftleftProjects: testutils.StringList(t),
		UserFilters:       testutils.StringList(t),
	}
	ua := &api_client.UserAccess{
		ID:                "assign-1",
		UserID:            "user-1",
		RoleID:            "role-1",
		CloudAccounts:     []string{},
		ShiftleftProjects: []string{},
		UserFilters:       []string{},
	}

	got, diags := r.apiToModel(context.Background(), ua, ref)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.CloudAccounts.IsNull() {
		t.Error("cloud_accounts should be empty non-null, got null")
	}
	var cloud []string
	if d := got.CloudAccounts.ElementsAs(context.Background(), &cloud, false); d.HasError() {
		t.Fatalf("elements as: %v", d)
	}
	if len(cloud) != 0 {
		t.Errorf("cloud_accounts should be empty, got %v", cloud)
	}
}

// modelToAPI -> apiToModel over a populated plan must preserve every field, so a
// Create/Update that echoes the payload produces state with no drift.
func TestModelToAPI_ApiToModel_RoundTrip(t *testing.T) {
	r := &userAccessResource{}
	plan := userAccessResourceModel{
		UserID:            types.StringValue("user-7"),
		RoleID:            types.StringValue("role-7"),
		AllCloudAccounts:  types.BoolValue(false),
		CloudAccounts:     testutils.StringList(t, "ca-a", "ca-b"),
		ShiftleftProjects: testutils.StringList(t, "sl-a"),
		UserFilters:       testutils.StringList(t, "bu-a"),
	}

	payload, diags := r.modelToAPI(context.Background(), plan, "assign-7")
	if diags.HasError() {
		t.Fatalf("modelToAPI diags: %v", diags)
	}
	state, diags := r.apiToModel(context.Background(), &payload, &plan)
	if diags.HasError() {
		t.Fatalf("apiToModel diags: %v", diags)
	}
	if state.ID.ValueString() != "assign-7" {
		t.Errorf("id drift: %q", state.ID.ValueString())
	}
	if state.UserID.ValueString() != "user-7" || state.RoleID.ValueString() != "role-7" {
		t.Errorf("scalar drift: %+v", state)
	}
	var cloud []string
	if d := state.CloudAccounts.ElementsAs(context.Background(), &cloud, false); d.HasError() {
		t.Fatalf("elements as: %v", d)
	}
	if !testutils.SameElements(cloud, []string{"ca-a", "ca-b"}) {
		t.Errorf("cloud accounts drifted: %v", cloud)
	}
}
