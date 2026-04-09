package group_access

import (
	"context"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMergeGroupAccessAfterCreate_PrefersRefreshed(t *testing.T) {
	refreshed := &api_client.GroupAccess{ID: "r1", GroupID: "g-remote", RoleID: "role1"}
	created := &api_client.GroupAccess{ID: "r1", GroupID: "g-created", RoleID: "role1"}
	plan := groupAccessResourceModel{}
	payload := api_client.GroupAccess{}

	out := mergeGroupAccessAfterCreate(refreshed, created, plan, payload)
	if out.GroupID != "g-remote" {
		t.Fatalf("expected remote group_id, got %q", out.GroupID)
	}
}

func TestMergeGroupAccessAfterCreate_FillsFromPlanAndPayload(t *testing.T) {
	created := &api_client.GroupAccess{
		ID:     "assign-1",
		RoleID: "role-x",
		// GroupID empty; nil slices
	}
	plan := groupAccessResourceModel{}
	plan.GroupID = types.StringValue("group-from-plan")
	plan.RoleID = types.StringValue("role-x")
	payload := api_client.GroupAccess{
		GroupID:           "group-from-plan",
		CloudAccounts:     []string{"ca1"},
		ShiftleftProjects: []string{"sl1"},
		UserFilters:       []string{"uf1"},
		AllCloudAccounts:  true,
	}

	out := mergeGroupAccessAfterCreate(nil, created, plan, payload)
	if out.GroupID != "group-from-plan" {
		t.Fatalf("GroupID: got %q", out.GroupID)
	}
	if out.RoleID != "role-x" {
		t.Fatalf("RoleID: got %q", out.RoleID)
	}
	if len(out.CloudAccounts) != 1 || out.CloudAccounts[0] != "ca1" {
		t.Fatalf("CloudAccounts: %+v", out.CloudAccounts)
	}
	if len(out.ShiftleftProjects) != 1 || out.ShiftleftProjects[0] != "sl1" {
		t.Fatalf("ShiftleftProjects: %+v", out.ShiftleftProjects)
	}
	if len(out.UserFilters) != 1 || out.UserFilters[0] != "uf1" {
		t.Fatalf("UserFilters: %+v", out.UserFilters)
	}
	if !out.AllCloudAccounts {
		t.Fatal("AllCloudAccounts should be true")
	}
}

func TestOptionalListMatchPlan_NullPlanEmptyAPI(t *testing.T) {
	ctx := context.Background()
	nullPlan := types.ListNull(types.StringType)
	got, diags := optionalListMatchPlan(ctx, nullPlan, nil)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !got.IsNull() {
		t.Fatalf("expected null list, got %#v", got)
	}
}

func TestOptionalListMatchPlan_NullPlanEmptySliceAPI(t *testing.T) {
	ctx := context.Background()
	nullPlan := types.ListNull(types.StringType)
	got, diags := optionalListMatchPlan(ctx, nullPlan, []string{})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !got.IsNull() {
		t.Fatalf("expected null list, got %#v", got)
	}
}

func TestOptionalListMatchPlan_UnknownPlanEmptyAPI(t *testing.T) {
	ctx := context.Background()
	unknownPlan := types.ListUnknown(types.StringType)
	got, diags := optionalListMatchPlan(ctx, unknownPlan, []string{})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !got.IsNull() {
		t.Fatalf("expected null list, got %#v", got)
	}
}

func TestOptionalListMatchPlan_EmptyKnownPlanEmptyAPI(t *testing.T) {
	ctx := context.Background()
	empty, diags := types.ListValueFrom(ctx, types.StringType, []string{})
	if diags.HasError() {
		t.Fatal(diags)
	}
	got, diags := optionalListMatchPlan(ctx, empty, []string{})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if got.IsNull() || got.IsUnknown() {
		t.Fatalf("expected known empty list, got %#v", got)
	}
	var elems []string
	_ = got.ElementsAs(ctx, &elems, false)
	if len(elems) != 0 {
		t.Fatalf("expected 0 elements, got %v", elems)
	}
}

func TestOptionalListMatchPlan_NonEmptyAPI(t *testing.T) {
	ctx := context.Background()
	nullPlan := types.ListNull(types.StringType)
	got, diags := optionalListMatchPlan(ctx, nullPlan, []string{"a", "b"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	var elems []string
	_ = got.ElementsAs(ctx, &elems, false)
	if len(elems) != 2 || elems[0] != "a" || elems[1] != "b" {
		t.Fatalf("got %v", elems)
	}
}
